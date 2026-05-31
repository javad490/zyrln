package tun

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	ipv4HeaderLen = 20
	tcpHeaderMin  = 20
)

type ipv4Packet struct {
	src     [4]byte
	dst     [4]byte
	proto   uint8
	payload []byte
}

func parseIPv4(raw []byte) (ipv4Packet, error) {
	if len(raw) < ipv4HeaderLen {
		return ipv4Packet{}, fmt.Errorf("short ipv4 packet")
	}
	if raw[0]>>4 != 4 {
		return ipv4Packet{}, fmt.Errorf("not ipv4")
	}
	ihl := int(raw[0]&0x0f) * 4
	if ihl < ipv4HeaderLen || len(raw) < ihl {
		return ipv4Packet{}, fmt.Errorf("bad ipv4 ihl")
	}
	totalLen := int(binary.BigEndian.Uint16(raw[2:4]))
	if totalLen < ihl || len(raw) < totalLen {
		return ipv4Packet{}, fmt.Errorf("bad ipv4 length")
	}
	var p ipv4Packet
	copy(p.src[:], raw[12:16])
	copy(p.dst[:], raw[16:20])
	p.proto = raw[9]
	p.payload = raw[ihl:totalLen]
	return p, nil
}

type tcpSegment struct {
	srcPort uint16
	dstPort uint16
	seq     uint32
	ack     uint32
	flags   uint8
	payload []byte
}

const (
	tcpFlagFIN = 0x01
	tcpFlagSYN = 0x02
	tcpFlagRST = 0x04
	tcpFlagPSH = 0x08
	tcpFlagACK = 0x10
)

func parseTCP(data []byte) (tcpSegment, error) {
	if len(data) < tcpHeaderMin {
		return tcpSegment{}, fmt.Errorf("short tcp segment")
	}
	off := int(data[12]>>4) * 4
	if off < tcpHeaderMin || len(data) < off {
		return tcpSegment{}, fmt.Errorf("bad tcp header")
	}
	var s tcpSegment
	s.srcPort = binary.BigEndian.Uint16(data[0:2])
	s.dstPort = binary.BigEndian.Uint16(data[2:4])
	s.seq = binary.BigEndian.Uint32(data[4:8])
	s.ack = binary.BigEndian.Uint32(data[8:12])
	s.flags = data[13]
	s.payload = data[off:]
	return s, nil
}

func ipToString(ip [4]byte) string {
	return net.IP(ip[:]).String()
}

func buildIPv4TCP(src, dst [4]byte, srcPort, dstPort uint16, seq, ack uint32, flags uint8, payload []byte) []byte {
	tcpLen := tcpHeaderMin + len(payload)
	total := ipv4HeaderLen + tcpLen
	buf := make([]byte, total)

	buf[0] = 0x45
	binary.BigEndian.PutUint16(buf[2:4], uint16(total))
	buf[8] = 64
	buf[9] = 6
	copy(buf[12:16], src[:])
	copy(buf[16:20], dst[:])
	binary.BigEndian.PutUint16(buf[10:12], ipChecksum(buf[:ipv4HeaderLen]))

	tcp := buf[ipv4HeaderLen:]
	binary.BigEndian.PutUint16(tcp[0:2], srcPort)
	binary.BigEndian.PutUint16(tcp[2:4], dstPort)
	binary.BigEndian.PutUint32(tcp[4:8], seq)
	binary.BigEndian.PutUint32(tcp[8:12], ack)
	tcp[12] = 0x50
	tcp[13] = flags
	binary.BigEndian.PutUint16(tcp[14:16], 65535)
	copy(tcp[tcpHeaderMin:], payload)
	binary.BigEndian.PutUint16(tcp[16:18], tcpChecksum(src, dst, tcp))
	return buf
}

type udpDatagram struct {
	srcPort uint16
	dstPort uint16
	payload []byte
}

func parseUDP(data []byte) (udpDatagram, error) {
	if len(data) < 8 {
		return udpDatagram{}, fmt.Errorf("short udp datagram")
	}
	length := int(binary.BigEndian.Uint16(data[4:6]))
	if length < 8 || len(data) < length {
		return udpDatagram{}, fmt.Errorf("bad udp length")
	}
	var d udpDatagram
	d.srcPort = binary.BigEndian.Uint16(data[0:2])
	d.dstPort = binary.BigEndian.Uint16(data[2:4])
	d.payload = data[8:length]
	return d, nil
}

func buildIPv4UDP(src, dst [4]byte, srcPort, dstPort uint16, payload []byte) []byte {
	udpLen := 8 + len(payload)
	total := ipv4HeaderLen + udpLen
	buf := make([]byte, total)

	buf[0] = 0x45
	binary.BigEndian.PutUint16(buf[2:4], uint16(total))
	buf[8] = 64
	buf[9] = 17
	copy(buf[12:16], src[:])
	copy(buf[16:20], dst[:])
	binary.BigEndian.PutUint16(buf[10:12], ipChecksum(buf[:ipv4HeaderLen]))

	udp := buf[ipv4HeaderLen:]
	binary.BigEndian.PutUint16(udp[0:2], srcPort)
	binary.BigEndian.PutUint16(udp[2:4], dstPort)
	binary.BigEndian.PutUint16(udp[4:6], uint16(udpLen))
	copy(udp[8:], payload)
	binary.BigEndian.PutUint16(udp[6:8], udpChecksum(src, dst, udp))
	return buf
}
