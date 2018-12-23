package transport

import (
	"net"

	"auth"
	"github.com/troian/easygo/netpoll"
	"systree"
)

// Conn is wrapper to net.Conn
// implemented to encapsulate bytes statistic
type Conn interface {
	net.Conn
	// Start(cb netpoll.CallbackFn) error
	// Stop() error
	// Resume() error
}

type conn struct {
	net.Conn
	stat systree.BytesMetric
	// desc  *netpoll.Desc
	// ePoll netpoll.EventPoll
}

var _ Conn = (*conn)(nil)

// Handler ...
type Handler interface {
	OnConnection(Conn, *auth.Manager) error
}

func newConn(poll netpoll.EventPoll, cn net.Conn, stat systree.BytesMetric) (*conn, error) {
	// desc, err := netpoll.HandleReadOnce(cn)
	// if err != nil {
	// 	return nil, err
	// }

	c := &conn{
		Conn: cn,
		stat: stat,
		// desc:  desc,
		// ePoll: poll,
	}

	return c, nil
}

// // Start ...
// func (c *conn) Start(cb netpoll.CallbackFn) error {
// 	return c.ePoll.Start(c.desc, cb)
// }
//
// // Stop ...
// func (c *conn) Stop() error {
// 	return c.ePoll.Stop(c.desc)
// }
//
// // Resume ...
// func (c *conn) Resume() error {
// 	return c.ePoll.Resume(c.desc)
// }

// Read ...
func (c *conn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)

	c.stat.Received(uint64(n))

	return n, err
}

// Write ...
func (c *conn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	c.stat.Sent(uint64(n))

	return n, err
}

// File ...
// func (c *conn) File() (*os.File, error) {
// 	switch t := c.Conn.(type) {
// 	case *net.TCPConn:
// 		return t.File()
// 	}
//
// 	return nil, errors.New("not implemented")
// }
