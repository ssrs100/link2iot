// +build !linux,!darwin,!dragonfly,!freebsd,!netbsd,!openbsd

package netpoll

import "fmt"

// New always returns an error to indicate that EventPoll is not implemented for
// current operating system.
func New(*Config) (EventPoll, error) {
	return nil, fmt.Errorf("poller is not supported on this operating system")
}
