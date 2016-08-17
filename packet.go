package main

import (
	"net"
)

const (
	PINGREQ  = 0xc0
	PINGRESP = 0xd0

	//RPC_PUBONL = 0x20
	//	RPC_PUBOFFL = 0x30
	//RPC_KICKOFF = 0x40

	RPC_PURE_PUB = 0x50 //只推送消息 不发送推送列表
	RPC_TONC     = 0x60 //统计在线用户
	RPC_TONC_ACK = 0x70 //统计在线用户答复
)

type Packet struct {
	fixHeader    uint8
	command      uint8
	remainLength uint32
	body         []byte
	bodyPos      uint32
}

func NewPacket(size uint32) *Packet {
	packet := &Packet{}
	packet.fixHeader = 0
	packet.command = 0
	packet.remainLength = 0
	packet.body = make([]byte, size)
	packet.bodyPos = 0
	return packet
}

func (self *Packet) readByte() byte {
	b := self.body[self.bodyPos]
	self.bodyPos += 1
	return b
}

func (self *Packet) readInt32() uint32 {
	a := uint32(self.body[self.bodyPos])
	self.bodyPos += 1
	b := uint32(self.body[self.bodyPos])
	self.bodyPos += 1
	c := uint32(self.body[self.bodyPos])
	self.bodyPos += 1
	d := uint32(self.body[self.bodyPos])
	self.bodyPos += 1

	return (a << 24) + (b << 16) + (c << 8) + d
}

func (self *Packet) writeByte(val byte) {
	self.body[self.bodyPos] = val
	self.bodyPos += 1
}

func (self *Packet) writeInt16(val uint16) {
	msb := byte((val & 0xFF00) >> 8)
	lsb := byte(val & 0x00FF)
	self.writeByte(msb)
	self.writeByte(lsb)
}

func (self *Packet) writeBytes(val []byte, length int) {
	copy(self.body[self.bodyPos:], val)
	self.bodyPos += uint32(length)
}

func (self *Packet) writeString(val string, length int) {
	self.writeInt16(uint16(length))
	self.writeBytes([]byte(val), length)

}

func ReceivePacket(conn net.Conn) (*Packet, error) {
	packet := &Packet{}
	packet.bodyPos = 0

	header := make([]byte, 1)
	if _, err := conn.Read(header); err != nil {
		return nil, err
	}
	packet.fixHeader = header[0]
	packet.command = header[0] & 0xF0

	multier := 1

	for {
		buf := make([]byte, 1)
		if _, err := conn.Read(buf); err != nil {
			return nil, err
		}
		packet.remainLength += uint32((int(buf[0]) & 127) * multier)
		multier *= 128

		if buf[0]&128 == 0 {
			break
		}
	}

	if packet.remainLength > 0 {
		packet.body = make([]byte, packet.remainLength)

		if _, err := conn.Read(packet.body); err != nil {
			return nil, err
		}
	}

	return packet, nil

}

func SendPacket(conn net.Conn, packet *Packet) error {
	buf := make([]byte, 1)
	buf[0] = packet.fixHeader

	length := packet.remainLength

	if length == 0 {
		buf = append(buf, 0)
	} else {

		for {
			digit := length % 128
			length = length / 128
			if length > 0 {
				digit = digit | 0x80
			}

			buf = append(buf, byte(digit))

			if length <= 0 {
				break
			}
		}

		buf = append(buf, packet.body...)
	}

	haveWrite := 0
	total := len(buf)
	for haveWrite < total {
		curLen, err := conn.Write(buf[haveWrite:])
		if nil != err {
			return err
		}
		haveWrite += curLen
	}

	return nil

}

func GainPingPacket() *Packet {
	packet := NewPacket(0)
	packet.remainLength = 0
	packet.command = PINGREQ
	packet.fixHeader = packet.command

	return packet
}

func GainPureMsgPacket(publishId string, topic string, message string) *Packet {

	remainLength := 2 + len(publishId) +
		2 + len(topic) +
		2 + len(message)

	packet := NewPacket(uint32(remainLength))
	packet.remainLength = uint32(remainLength)
	packet.command = RPC_PURE_PUB
	packet.fixHeader = packet.command

	packet.writeString(publishId, len(publishId))
	packet.writeString(topic, len(topic))
	packet.writeString(message, len(message))
	return packet
}
func GainQueryOnlinePacket(topic string) *Packet {

	remainLength := 2 + len(topic)

	packet := NewPacket(uint32(remainLength))
	packet.remainLength = uint32(remainLength)
	packet.command = RPC_TONC
	packet.fixHeader = packet.command

	packet.writeString(topic, len(topic))
	return packet
}
