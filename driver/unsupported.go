// +build windows

package kvm

import (
	"fmt"

	"github.com/virtmonitor/driver"
)

// Detect Detect dependencies
func (k *KVM) Detect() bool {
	return false
}

// Collect Collect domain statistics
func (k *KVM) Collect(cpu bool, block bool, network bool) (domains map[driver.DomainID]*driver.Domain, err error) {
	domains = make(map[driver.DomainID]*driver.Domain)
	err = fmt.Errorf("KVM not supported on this platform")
	return
}
