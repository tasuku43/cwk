//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || zos

package terminalui

import (
	"context"
	"fmt"
	"io"

	"golang.org/x/sys/unix"
)

func readTerminal(ctx context.Context, fd int, buffer []byte) (int, error) {
	if fd < 0 || int64(fd) > 1<<31-1 {
		return 0, fmt.Errorf("invalid terminal input descriptor %d", fd)
	}
	// #nosec G115 -- the explicit non-negative int32 bound above precedes the
	// only narrowing conversion accepted by PollFd.
	poll := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		ready, err := unix.Poll(poll, terminalReadWaitMilliseconds)
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return 0, err
		}
		// Cancellation wins after every readiness wait. In particular, a byte
		// that arrives after cancellation remains available for the next owner.
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if ready == 0 {
			continue
		}
		events := poll[0].Revents
		if events&unix.POLLNVAL != 0 {
			return 0, fmt.Errorf("terminal input descriptor is invalid")
		}
		if events&(unix.POLLIN|unix.POLLHUP|unix.POLLERR) == 0 {
			continue
		}
		count, err := unix.Read(fd, buffer)
		if err == unix.EINTR || err == unix.EAGAIN {
			continue
		}
		if err != nil {
			return count, err
		}
		if count == 0 {
			return 0, io.EOF
		}
		return count, nil
	}
}
