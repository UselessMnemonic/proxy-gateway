package api

import (
	"fmt"

	"golang.org/x/sys/unix"
)

type Protocol uint8

const (
	ProtocolTCP Protocol = unix.IPPROTO_TCP
	ProtocolUDP Protocol = unix.IPPROTO_UDP
)

func (p Protocol) String() string {
	switch p {
	case ProtocolTCP:
		return "tcp"
	case ProtocolUDP:
		return "udp"
	default:
		return "invalid"
	}
}

func (p Protocol) IsValid() bool {
	return p.String() != "invalid"
}

func ParseProtocol(s string) (Protocol, error) {
	switch s {
	case "tcp":
		return ProtocolTCP, nil
	case "udp":
		return ProtocolUDP, nil
	default:
		return 0, fmt.Errorf("invalid Protocol %q", s)
	}
}

func (p Protocol) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *Protocol) UnmarshalText(text []byte) error {
	result, err := ParseProtocol(string(text))
	if err != nil {
		return err
	}
	*p = result
	return nil
}
