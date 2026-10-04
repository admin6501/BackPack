package l3

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
)

// FOU carries protocol 47 directly in UDP, without a FOU-specific header.
// The GRE payload here is Backpack's authenticated Noise datagram, identified
// by the local experimental EtherType 0x88b5. It is deliberately not labelled
// IPv4: encrypted bytes are not an IP packet. Both ends must run Backpack;
// this is not a plaintext Linux GRE tunnel.
const fouProtocol uint16 = 0x88b5

type fouCarrier struct {
	net.PacketConn
	overhead int
	readMu   sync.Mutex
	readBuf  [65536]byte
	writeMu  sync.Mutex
	writeBuf [65507]byte
}

func (c *fouCarrier) CarrierName() string { return CarrierGREFOU }
func (c *fouCarrier) Overhead() int       { return c.overhead }

func openGREFOU(cfg Config) (DatagramCarrier, net.Addr, error) {
	var conn DatagramCarrier
	var peer net.Addr
	var err error
	if cfg.Mode == ModeDial {
		conn, peer, err = dialUDP(cfg.Addr, cfg.SockBuf)
	} else {
		conn, err = listenUDP(cfg.Addr, cfg.SockBuf)
	}
	if err != nil {
		return nil, nil, err
	}
	return &fouCarrier{PacketConn: conn, overhead: conn.Overhead() + 4}, peer, nil
}

func (c *fouCarrier) WriteTo(p []byte, addr net.Addr) (int, error) {
	if len(p) > len(c.writeBuf)-4 {
		return 0, fmt.Errorf("gre-fou: payload exceeds UDP datagram limit")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	frame := c.writeBuf[:4+len(p)]
	binary.BigEndian.PutUint16(frame[2:4], fouProtocol)
	copy(frame[4:], p)
	n, err := c.PacketConn.WriteTo(frame, addr)
	if err != nil {
		return 0, err
	}
	if n != len(frame) {
		return 0, io.ErrShortWrite
	}
	return len(p), nil
}

func (c *fouCarrier) ReadFrom(p []byte) (int, net.Addr, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	for {
		n, addr, err := c.PacketConn.ReadFrom(c.readBuf[:])
		if err != nil {
			return 0, addr, err
		}
		// Reject unsupported GRE flags/version and protocols before handing any
		// data to the session parser. Deadline/Close remain effective in this loop.
		if n < 4 || binary.BigEndian.Uint16(c.readBuf[:2]) != 0 ||
			binary.BigEndian.Uint16(c.readBuf[2:4]) != fouProtocol {
			continue
		}
		if n-4 > len(p) {
			continue
		}
		return copy(p, c.readBuf[4:n]), addr, nil
	}
}
