package main

import (
	"runtime"
	"syscall"
)

// Setuid sets uid of currect thread. For linux where syscall.Setuid isn't supported it calls raw syscall.
func Setuid(uid int) error {
	err := syscall.Setuid(uid)
	if err == syscall.EOPNOTSUPP && runtime.GOOS == "linux" {
		_, _, errno := syscall.RawSyscall(syscall.SYS_SETUID, uintptr(uid), 0, 0)
		if errno == 0 {
			err = nil
		} else {
			err = errno
		}
	}
	return err
}

// Setgid sets gid of currect thread. For linux where syscall.Setgid isn't supported it calls raw syscall.
func Setgid(gid int) error {
	err := syscall.Setgid(gid)
	if err == syscall.EOPNOTSUPP && runtime.GOOS == "linux" {
		_, _, errno := syscall.RawSyscall(syscall.SYS_SETGID, uintptr(gid), 0, 0)
		if errno == 0 {
			err = nil
		} else {
			err = errno
		}
	}
	return err
}
