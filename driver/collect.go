// +build !windows

package kvm

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/digitalocean/go-qemu/qemu"
	"github.com/digitalocean/go-qemu/qmp"
	gopsutil "github.com/shirou/gopsutil/process"
	"github.com/virtmonitor/driver"
	"github.com/virtmonitor/virNetTap"
)

type QemuProcess struct {
	id      uint64
	uuid    string
	name    string
	qmp     string
	ifnames []string
}

type QemuCPU struct {
	CPU      int   `json:"cpu"`
	Current  bool  `json:"current"`
	Halted   bool  `json:"halted"`
	PC       int   `json:"pc"`
	ThreadID int32 `json:"thread_id"`
}

// Collect Collect domain statistics
func (k *KVM) Collect(cpu bool, block bool, network bool) (domains map[driver.DomainID]*driver.Domain, err error) {
	//log.Println("KVM Collect()")
	domains = make(map[driver.DomainID]*driver.Domain)

	var processes []*gopsutil.Process
	if processes, err = gopsutil.Processes(); err != nil {
		err = fmt.Errorf("Could not gather process list: %v", err)
		return
	}

	var qprocesses []QemuProcess

	var process *gopsutil.Process
	var cmdline []string
	for _, process = range processes {
		var qprocess QemuProcess
		if cmdline, err = process.CmdlineSlice(); err == nil {
			if len(cmdline) <= 0 {
				continue
			}

			//log.Println(cmdline[0])
			//if !filepath.HasPrefix(cmdline[0], "/usr/bin") || !filepath.HasPrefix(cmdline[0], "/bin") {
			//	continue
			//}

			base := filepath.Base(cmdline[0])
			for _, binary := range qemuBinays {
				if strings.Contains(base, binary) {
					id, name, qmp, uuid, ifnames, err := parseCmdline(cmdline)
					if err != nil {
						log.Printf("Error parsing command line: %v", err)
						break
					}

					qprocess.id = *id
					qprocess.name = *name
					qprocess.uuid = uuid
					qprocess.qmp = *qmp
					qprocess.ifnames = ifnames

					log.Printf("qprocess: %+v\r\n", qprocess)

					qprocesses = append(qprocesses, qprocess)
					break
				}
			}
		}
	}

	var procnet virNetTap.VirNetTap
	nstat := make(map[string]virNetTap.InterfaceStats)

	if nstat, err = procnet.GetAllVifStats(); err != nil {
		return
	}

	var qprocess QemuProcess
	for _, qprocess = range qprocesses {
		domain := &driver.Domain{}
		var socket *qmp.SocketMonitor

		if qprocess.qmp == "" {
			log.Printf("Could not locate QMP monitor for domain #%d", qprocess.id)
			continue
		} else {
			fields := strings.Split(qprocess.qmp, ":")
			if socket, err = qmp.NewSocketMonitor(fields[0], strings.Join(fields[1:], ":"), 100*time.Millisecond); err != nil {
				log.Printf("Error opening socket monitor for domain #%d: %v", qprocess.id, err)
				continue
			}
		}

		if err = socket.Connect(); err != nil {
			log.Printf("Could not connect to qmp socket %s: %v", qprocess.qmp, err)
			continue
		}
		defer socket.Disconnect()

		var qdomain *qemu.Domain
		if qdomain, err = qemu.NewDomain(socket, qprocess.name); err != nil {
			log.Printf("Could not attach to domain from qmp monitor %s: %v", qprocess.qmp, err)
			continue
		}

		domain.Name = qdomain.Name
		domain.UUID = qprocess.uuid
		domain.ID = driver.DomainID(qprocess.id)

		if cpu {
			var raw []byte
			if raw, err = qdomain.Run(qmp.Command{Execute: "query-cpus"}); err != nil {
				log.Printf("Error querying cpus from qmp: %v", err)
				continue
			}

			var response struct {
				Return []QemuCPU `json:"return`
			}

			if err = json.Unmarshal(raw, &response); err != nil {
				log.Printf("Error unmarshaling query-cpu response: %v", err)
				continue
			}

			for _, cpu := range response.Return {
				var dcpu driver.CPU
				dcpu.ID = uint64(cpu.CPU)
				if cpu.Current {
					dcpu.Flags |= driver.CPUOnline
				}
				if cpu.Halted {
					dcpu.Flags |= driver.CPUHalted
				}
				//read schedstats for thread_id
				var data []byte
				if data, err = ioutil.ReadFile(fmt.Sprintf("/proc/%d/schedstat", cpu.ThreadID)); err != nil {
					log.Printf("Could not read schedstat for domain #%d cpu #%d: %v", qprocess.id, cpu, err)
				} else {
					fields := strings.Fields(string(data))

					if len(fields) < 3 {
						log.Printf("Error parsing schedstats for domain #%d cpu #%d: Expected 3 fields got %d", qprocess.id, cpu, len(fields))
					}

					if dcpu.Time, err = strconv.ParseFloat(fields[0], 64); err != nil {
						log.Printf("Error parsing cputime for domain #%d cpu %d: Could not convert string to float", qprocess.id, cpu)
					} else {
						dcpu.Time = dcpu.Time / float64(1000000000)
					}

					domain.Cpus = append(domain.Cpus, dcpu)
				}
			}
		}

		if block {
			var qdevices []qemu.BlockDevice
			var qblocks []qemu.BlockStats

			if qdevices, err = qdomain.BlockDevices(); err != nil {
				log.Printf("Error gathering blockdevices for domain #%d: %v", qprocess.id, err)
			} else {

				if qblocks, err = qdomain.BlockStats(); err != nil {
					log.Printf("Error gathering blockstats for domain #%d: %v", qprocess.id, err)
				} else {

					for _, block := range qblocks {
						var dblock driver.BlockDevice

						dblock.Name = block.Device

						for _, device := range qdevices {

							if device.Device == block.Device {
								dblock.ReadOnly = device.Inserted.ReadOnly

								//log.Printf("Device: %+v\r\n", device)

								//if strings.ToLower(filepath.Ext(device.Inserted.File)) == "iso" || device.Inserted.Driver == "host_cdrom" || strings.Contains(device.Device, "cd") || device.Type == "cdrom" {
								//	dblock.IsCDrom = true
								//} else if (strings.Contains(device.Device, "hd") || strings.Contains(device.Device, "sd") || device.Type == "hd" || device.Type == "sd") || (device.Inserted.Driver != "host_floppy" && !strings.HasPrefix(device.Inserted.Driver, "http") && !strings.HasPrefix(device.Inserted.Driver, "ftp")) {
								dblock.IsDisk = true
								//}

								dblock.IsCDrom, dblock.IsDisk = deviceType(device)
							}
						}

						dblock.Flush.Operations = uint64(block.FlushOperations)

						dblock.Write.Operations = uint64(block.WriteOperations)
						dblock.Write.Sectors = block.WriteBytes / 512
						dblock.Write.Bytes = block.WriteBytes

						dblock.Read.Operations = uint64(block.ReadOperations)
						dblock.Read.Sectors = block.ReadBytes / 512
						dblock.Read.Bytes = block.ReadBytes

						domain.Blocks = append(domain.Blocks, dblock)
					}
				}

			}
		}

		if network {

			for _, ifname := range qprocess.ifnames {
				var dnetwork driver.NetworkInterface
				dnetwork.Name = ifname

				var bridges []string
				if bridges, err = filepath.Glob(fmt.Sprintf("/sys/class/net/%s/upper_*", ifname)); err == nil && len(bridges) > 0 {
					for _, bridge := range bridges {
						dnetwork.Bridges = append(dnetwork.Bridges, strings.ToLower(strings.TrimPrefix(filepath.Base(bridge), "upper_")))
					}
				}

				//TODO: parse out the mac address for interface from cmdline arguments?

				if vifstat, ok := nstat[ifname]; ok {
					dnetwork.RX = driver.NetworkIO{Bytes: vifstat.IN.Bytes, Packets: vifstat.IN.Pkts, Errors: vifstat.IN.Errs, Drops: vifstat.IN.Drops}
					dnetwork.TX = driver.NetworkIO{Bytes: vifstat.OUT.Bytes, Packets: vifstat.OUT.Pkts, Errors: vifstat.OUT.Errs, Drops: vifstat.OUT.Drops}
				}

				domain.Interfaces = append(domain.Interfaces, dnetwork)

			}

		}

		//log.Printf("Domain: %+v", domain)
		domains[domain.ID] = domain
		//log.Printf("%d", len(domains))

	}

	return
}

func deviceType(device qemu.BlockDevice) (cdrom bool, disk bool) {

	if device.Type == "cdrom" || strings.Contains(device.Device, "cd") {
		cdrom = true
		return
	} else if device.Type == "hd" || device.Type == "sd" || strings.Contains(device.Device, "hd") || strings.Contains(device.Device, "sd") {
		disk = true
		return
	}

	switch device.Inserted.Driver {
	case "host_cdrom":
		cdrom = true
		return
	case "host_floppy", "http", "https", "ftp":
		break
	default:
		disk = true
		return
	}

	if strings.HasSuffix(device.Inserted.File, ".iso") || strings.HasSuffix(device.Inserted.BackingFile, ".iso") {
		cdrom = true
		return
	} else if strings.HasSuffix(device.Inserted.Image.BackingFilename, ".iso") || strings.HasSuffix(device.Inserted.Image.Filename, ".iso") {
		cdrom = true
		return
	} else if strings.HasSuffix(device.Inserted.Image.BackingImage.Filename, ".iso") {
		cdrom = true
		return
	}

	return
}
