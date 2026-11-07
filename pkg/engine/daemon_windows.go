package engine

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

const (
	jobObjectExtendedLimitInformation = 9
	jobObjectLimitKillOnJobClose      = 0x2000
	processSetQuotaAndTerminate       = 0x0100 | 0x0001
)

// jobObjectExtendedLimit is JOBOBJECT_EXTENDED_LIMIT_INFORMATION on 64-bit Windows.
type jobObjectExtendedLimit struct {
	perProcessUserTimeLimit int64
	perJobUserTimeLimit     int64
	limitFlags              uint32
	minimumWorkingSetSize   uintptr
	maximumWorkingSetSize   uintptr
	activeProcessLimit      uint32
	affinity                uintptr
	priorityClass           uint32
	schedulingClass         uint32
	ioCounters              [6]uint64
	processMemoryLimit      uintptr
	jobMemoryLimit          uintptr
	peakProcessMemoryUsed   uintptr
	peakJobMemoryUsed       uintptr
}

var daemonJob = sync.OnceValues(func() (syscall.Handle, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	h, _, err := kernel32.NewProc("CreateJobObjectW").Call(0, 0)
	if h == 0 {
		return 0, fmt.Errorf("CreateJobObject: %w", err)
	}
	info := jobObjectExtendedLimit{limitFlags: jobObjectLimitKillOnJobClose}
	ok, _, err := kernel32.NewProc("SetInformationJobObject").Call(h, jobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info))
	if ok == 0 {
		return 0, fmt.Errorf("SetInformationJobObject: %w", err)
	}
	return syscall.Handle(h), nil // held for the life of the process; the OS closing it kills the job
})

// killWithParent puts cmd in a job that Windows kills when this process ends, however it ends.
func killWithParent(cmd *exec.Cmd) error {
	job, err := daemonJob()
	if err != nil {
		return err
	}
	proc, err := syscall.OpenProcess(processSetQuotaAndTerminate, false, uint32(cmd.Process.Pid))
	if err != nil {
		return fmt.Errorf("OpenProcess: %w", err)
	}
	defer syscall.CloseHandle(proc)
	ok, _, err := syscall.NewLazyDLL("kernel32.dll").NewProc("AssignProcessToJobObject").Call(uintptr(job), uintptr(proc))
	if ok == 0 {
		return fmt.Errorf("AssignProcessToJobObject: %w", err)
	}
	return nil
}
