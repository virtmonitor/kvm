package kvm

import "github.com/virtmonitor/driver"

var (
	qemuBinays = []string{
		"kvm",
		"qemu-system-x86_64",
		"qemu-system-x86",
	}

	qmpPaths = []string{
		"/var/run/qemu",
	}
)

//KVM KVM struct
type KVM struct {
	driver.Driver
}

//Name Return driver name
func (k *KVM) Name() driver.DomainHypervisor {
	return "KVM"
}

//Close Close driver
func (k *KVM) Close() {
	return
}
