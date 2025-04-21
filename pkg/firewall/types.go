package firewall

import (
	"encoding/binary"

	"golang.org/x/sys/unix"
)

// host to network byte order short uint
func htons(hostshort uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, hostshort)
	return b
}

type ExprToDataable interface {
	ToData() []byte
}

type L4Proto int

const (
	UDP = unix.IPPROTO_UDP
	TCP = unix.IPPROTO_TCP
)

func (l L4Proto) ToData() []byte {
	// less than 255, direct to byte
	return []byte{byte(l)}
}
