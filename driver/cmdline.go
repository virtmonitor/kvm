package kvm

import (
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
	opt.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}

	id = opt.Uint64("id", 0, "")
	name = opt.String("name", "", "")
	qmp = opt.String("qmp", "", "")
	smbios := opt.String("smbios", "", "")

	var chardev, netdev, mon stringList
	opt.Var(&chardev, "chardev", "")
	opt.Var(&netdev, "netdev", "")
	opt.Var(&mon, "mon", "")

	// pflag does not like longhand (--) flags having a shorthand delim (-)

	/*var args2 []string

	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			args2 = append(args2, arg)
			continue
		}
		if strings.HasPrefix(arg, "-") {
			if len(arg) == 2 {
				//shorthand
				args2 = append(args2, arg)
				continue
			}
			//longhand with shorthand delim
			args2 = append(args2, "-"+arg)
			continue
		}
		args2 = append(args2, arg)
	}

	args = args2

	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			arg = strings.TrimPrefix(arg, "--")
		} else if strings.HasPrefix(arg, "-") {
			arg = strings.TrimPrefix(arg, "-")
		} else {
			continue
		}
		switch arg {
		case "id", "name", "qmp", "mon", "chardev", "netdev", "smbios":
			continue
		case "nodefaults", "daemonize", "S", "enable-kvm":
			if len(arg) == 1 && opt.ShorthandLookup(arg) == nil {
				log.Println("BoolP:", arg)
				_ = opt.BoolP("", arg, false, "")
			} else if opt.Lookup(arg) == nil {
				log.Println("Bool:", arg)
				_ = opt.Bool(arg, false, "")
			}
		case "m":
			if len(arg) == 1 && opt.ShorthandLookup(arg) == nil {
				log.Println("IntP:", arg)
				_ = opt.IntP("", arg, 0, "")
			} else if opt.Lookup(arg) == nil {
				log.Println("Int:", arg)
				_ = opt.Int(arg, 0, "")
			}
		default:
			if len(arg) == 1 && opt.ShorthandLookup(arg) == nil {
				log.Println("StringP:", arg)
				_ = opt.StringP("", arg, "", "")
			} else if opt.Lookup(arg) == nil {
				log.Println("String:", arg)
				_ = opt.String(arg, "", "")
			}
		}
	}*/

	err = opt.Parse(args[1:])
	if err != nil {
		if err = pflag.ErrHelp {
			err = nil
			log.Println("ErrHelp blah blah")
		} else {
			return
		}
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

/*func parseCmdline2(args []string) (id *uint64, name, qmp *string, uuid string, ifnames []string, err error) {

	opt := pflag.NewFlagSet(args[0], pflag.ContinueOnError)

	for _, arg := range args[1:] {
		if string.HasPrefix(arg, "--") {

		}
	}

	return
}*/
