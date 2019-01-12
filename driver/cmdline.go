// +build !windows

package kvm

import (
	"flag"
	"fmt"
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

	id = opt.Uint64("id", 0, "")
	name = opt.StringP("name", "name", "", "")
	qmp = opt.String("qmp", "", "")
	smbios := opt.String("smbios", "", "")

	var chardev, netdev, mon stringList
	opt.Var(&chardev, "chardev", "")
	opt.Var(&netdev, "netdev", "")
	opt.Var(&mon, "mon", "")

	//var acc bool
	//opt.Bool(&acc, false, "")

	//var args2 []string

	//for _, arg := range args {
	//	if strings.ToLower(arg) == "-enable-kvm" {
	//		continue
	//	}
	//	args2 = append(args2, arg)
	//}

	//args = args2

	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		arg = strings.TrimPrefix(arg, "-")
		switch arg {
		case "id", "name", "qmp", "mon", "chardev", "netdev", "smbios":
			continue
		case "nodefaults", "daemonize":
			if opt.Lookup(arg) == nil {
				_ = opt.Bool(arg, false, "")
			}
		default:
			if opt.Lookup(arg) == nil {
				_ = opt.String(arg, "", "")
			}
		}
	}

	if err = opt.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return
		}
		err = fmt.Errorf("Error parsing %s command line arguments: %v", args[0], err)
		return
	}

	var str string

	//parse uuid
	for _, str = range strings.Split(*smbios, ",") {
		if strings.HasPrefix(str, "uuid=") {
			uuid = strings.TrimPrefix(str, "uuid=")
			break
		}
	}

	//parse ifnames
	for _, str = range strings.Split(netdev.String(), ",") {
		if strings.HasPrefix(str, "ifname=") {
			ifnames = append(ifnames, strings.TrimPrefix(str, "ifname="))
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
