// +build !windows

package kvm

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Detect Detect dependencies
func (k *KVM) Detect() bool {
	var err error

	var f *os.File
	if f, err = os.Open("/proc/modules"); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.Printf("Error detecting KVM driver: %v", err)
		return false
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "kvm") {
			goto detect_binarys
		}
	}

	return false

detect_binarys:

	for _, binary := range qemuBinays {
		if _, err = exec.LookPath(binary); err == nil {
			return true
		}
	}

	return false
}
