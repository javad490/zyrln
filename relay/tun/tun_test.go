package tun

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestParseBuildIPv4TCP(t *testing.T) {
	var src, dst [4]byte
	copy(src[:], net.ParseIP("10.99.0.2").To4())
	copy(dst[:], net.ParseIP("91.108.8.6").To4())

	payload := []byte("hello")
	raw := buildIPv4TCP(src, dst, 12345, 443, 100, 200, tcpFlagACK|tcpFlagPSH, payload)

	ip, err := parseIPv4(raw)
	if err != nil {
		t.Fatal(err)
	}
	if ip.proto != 6 {
		t.Fatalf("proto=%d", ip.proto)
	}
	seg, err := parseTCP(ip.payload)
	if err != nil {
		t.Fatal(err)
	}
	if seg.srcPort != 12345 || seg.dstPort != 443 {
		t.Fatalf("ports %d %d", seg.srcPort, seg.dstPort)
	}
	if string(seg.payload) != "hello" {
		t.Fatalf("payload=%q", seg.payload)
	}
}

func TestIPChecksum(t *testing.T) {
	hdr := make([]byte, 20)
	hdr[0] = 0x45
	binary.BigEndian.PutUint16(hdr[2:4], 20)
	hdr[9] = 6
	copy(hdr[12:16], net.ParseIP("10.0.0.1").To4())
	copy(hdr[16:20], net.ParseIP("10.0.0.2").To4())
	c := ipChecksum(hdr)
	if c == 0 {
		t.Fatal("expected non-zero checksum")
	}
}

func TestTCPMSS(t *testing.T) {
	if tcpMSS <= 0 || tcpMSS > 1460 {
		t.Fatalf("tcpMSS=%d", tcpMSS)
	}
	if maxPendingBytes <= tcpMSS {
		t.Fatalf("maxPendingBytes=%d", maxPendingBytes)
	}
}

func TestFlowKeyTarget(t *testing.T) {
	var dst [4]byte
	copy(dst[:], net.ParseIP("1.2.3.4").To4())
	k := flowKey{dst: dst, dstPort: 443}
	if k.target() != "1.2.3.4:443" {
		t.Fatalf("target=%q", k.target())
	}
}
