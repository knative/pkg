package spoof

import (
	"net"
	"net/url"
	"strings"
)

func isTCPTimeout(e error) bool {
	err, ok := e.(net.Error)
	return err != nil && ok && err.Timeout()
}

func isDNSError(err error) bool {
	if err, ok := err.(*url.Error); err != nil && ok {
		if err, ok := err.Err.(*net.OpError); err != nil && ok {
			if err, ok := err.Err.(*net.DNSError); err != nil && ok {
				return true
			}
		}
	}
	return false
}

func isTCPConnectRefuse(err error) bool {
	// The alternative for the string check is:
	// 	errNo := (((err.(*url.Error)).Err.(*net.OpError)).Err.(*os.SyscallError).Err).(syscall.Errno)
	// if errNo == syscall.Errno(0x6f) {...}
	// But with assertions, of course.
	if err != nil && strings.Contains(err.Error(), "connect: connection refused") {
		return true
	}
	return false
}
