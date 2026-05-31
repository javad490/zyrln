package tun

import (
	"context"
	"net"
	"time"

	"zyrln/relay/netdial"
)

const (
	dnsTimeout   = 5 * time.Second
	maxDNSInflight = 32
)

func (e *Engine) handleDNS(ip ipv4Packet, d udpDatagram) {
	if d.dstPort != 53 || len(d.payload) == 0 {
		return
	}
	select {
	case e.dnsSem <- struct{}{}:
	default:
		return
	}
	query := append([]byte(nil), d.payload...)
	go e.resolveDNS(ip, d, query)
}

func (e *Engine) resolveDNS(ip ipv4Packet, d udpDatagram, query []byte) {
	defer func() { <-e.dnsSem }()

	select {
	case <-e.ctx.Done():
		return
	default:
	}

	target := net.JoinHostPort(ipToString(ip.dst), "53")
	ctx, cancel := context.WithTimeout(e.ctx, dnsTimeout)
	defer cancel()

	conn, err := netdial.ProtectedDialer(dnsTimeout).DialContext(ctx, "udp", target)
	if err != nil {
		return
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(dnsTimeout))
	if _, err := conn.Write(query); err != nil {
		return
	}
	resp := make([]byte, 4096)
	n, err := conn.Read(resp)
	if err != nil || n == 0 {
		return
	}
	pkt := buildIPv4UDP(ip.dst, ip.src, d.dstPort, d.srcPort, resp[:n])
	e.writeTUN(pkt)
}
