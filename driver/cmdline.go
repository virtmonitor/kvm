package kvm

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/pflag"
)

type stringList []string

func (l *stringList) String() string {
	return fmt.Sprintf("%s", *l)
}

func (l *stringList) Set(value string) error {
	*l = append(*l, value)
	return nil
}

func (l *stringList) Type() string {
	return "string"
}

func parseCmdline(args []string) (id *uint64, name, qmp *string, uuid string, ifnames []string, err error) {

	opt := pflag.NewFlagSet(args[0], pflag.ContinueOnError)
	opt.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}

	id = opt.Uint64("id", 0, "")
	name = opt.String("name", "", "")
	uuid = *opt.String("uuid", "", "")
	qmp = opt.String("qmp", "", "")
	smbios := opt.String("smbios", "", "")

	var chardev, netdev, mon, device stringList
	opt.Var(&chardev, "chardev", "")
	opt.Var(&netdev, "netdev", "")
	opt.Var(&device, "device", "")
	opt.Var(&mon, "mon", "")

	var args2 []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			args2 = append(args2, arg)
		} else if strings.HasPrefix(arg, "-") {
			args2 = append(args2, "-"+arg)
		} else {
			args2 = append(args2, arg)
		}
	}

	err = opt.Parse(args2[1:])
	if err != nil {
		if err == pflag.ErrHelp {
			err = nil
			log.Println("ErrHelp blah blah")
		} else {
			return
		}
	}

	var str string

	//{id:0 uuid: name:d9 qmp:unix:/var/lib/libvirt/qemu/d9.monitor ifnames:[]}

	//parse uuid
	if uuid == "" {
		for _, str = range strings.Split(*smbios, ",") {
			if strings.HasPrefix(str, "uuid=") {
				uuid = strings.TrimPrefix(str, "uuid=")
				break
			}
		}
	}

	//parse ifnames
	for _, str = range strings.Split(netdev.String(), ",") {
		if strings.HasPrefix(str, "ifname=") {
			ifnames = append(ifnames, strings.TrimPrefix(str, "ifname="))
		}
	}
	for _, str = range strings.Split(device.String(), ",") {
		if strings.HasPrefix("str", "id=") {
			ifnames = append(ifnames, "v"+strings.TrimPrefix(str, "id="))
		}
	}

	//parse qmp
	if *qmp != "" {
		for _, str = range strings.Split(*qmp, ",") {
			if strings.HasPrefix(str, "unix:") || strings.HasPrefix(str, "tcp:") {
				*qmp = str
				break
			}
		}
	} else if len(chardev) > 0 && len(mon) > 0 {
		for _, monitor := range mon {
			m := toMap(monitor, ",", "=")

			if mode, ok := m["mode"]; ok {
				if mode == "control" {
					for _, char := range chardev {
						mm := toMap(char, ",", "=")

						if mm["id"] == m["chardev"] {
							if path, ok := mm["path"]; ok {
								*qmp = fmt.Sprintf("unix:%s", path)
								break
							} else if host, ok := mm["host"]; ok {
								if port, ok := mm["port"]; ok {
									*qmp = fmt.Sprintf("tcp:%s:%s", host, port)
									break
								}
							}
						}
					}
				}
			}
		}

	}
	return
}

func toMap(entry, delim1, delim2 string) map[string]string {
	m := make(map[string]string)

	for _, str := range strings.Split(entry, delim1) {

		fields := strings.Split(str, delim2)

		if len(fields) == 1 {
			m[fields[0]] = ""
		} else if len(fields) > 1 {
			m[fields[0]] = strings.Join(fields[1:], delim1)
		}

	}

	return m
}
