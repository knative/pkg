package spoof

import (
	"net"
	"net/url"
	"strings"
)

func isTCPTimeout(err error) bool {
	if err, ok := err.(net.Error); ok && err.Timeout() {
		return true
	}
	return false
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
	if strings.Contains(err.Error(), "connect: connection refused") {
		return true
	}
	return false
}
