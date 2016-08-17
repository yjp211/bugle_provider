package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type BrokerConn struct {
	addr     string
	Using    bool
	conn     net.Conn
	timeout  int
	writeErr bool
}

func NewBrokerConn(addr string, timeout int) *BrokerConn {
	client := &BrokerConn{}
	client.addr = addr
	client.timeout = timeout
	//	tcpAddr, err := net.ResolveTCPAddr("tcp4", addr)
	//	if err != nil {
	//		log.Error("conncet to <%s> failed, %v", addr, err)
	//		return nil
	//	}
	//	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	duration := time.Second * time.Duration(timeout)
	conn, err := net.DialTimeout("tcp", addr, duration)
	if err != nil {
		log.Error("conncet to <%s> failed, %v", addr, err)
		return nil
	}
	client.conn = conn
	client.writeErr = false
	return client
}

func (self *BrokerConn) IsActive() bool {
	if self.writeErr {
		return false
	}

	return self.SendPing()
}

func (self *BrokerConn) SendPing() bool {
	packet := GainPingPacket()
	if self.timeout > 0 {
		self.conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(self.timeout)))
	}
	err := SendPacket(self.conn, packet)
	if nil != err {
		self.writeErr = true
		return false
	}
	if self.timeout > 0 {
		self.conn.SetReadDeadline(time.Now().Add(time.Second *
			time.Duration(self.timeout)))
	}

	packet, err = ReceivePacket(self.conn)
	if nil != err || packet.command != PINGRESP {
		self.writeErr = true
		return false
	}

	return true
}

func (self *BrokerConn) PublishPureMsg(publishId string, topic string, message string) error {
	packet := GainPureMsgPacket(publishId, topic, message)
	err := SendPacket(self.conn, packet)
	if nil != err {
		self.writeErr = true
	}
	return err
}

func (self *BrokerConn) QueryTopicOnline(topic string) (int64, error) {
	packet := GainQueryOnlinePacket(topic)
	err := SendPacket(self.conn, packet)
	if nil != err {
		self.writeErr = true
		return 0, err
	}

	if self.timeout > 0 {
		self.conn.SetReadDeadline(time.Now().Add(time.Second *
			time.Duration(self.timeout)))
	}

	packet, err = ReceivePacket(self.conn)
	if nil != err || packet.command != RPC_TONC_ACK {
		self.writeErr = true
		return 0, err
	}

	online := int64(packet.readInt32())

	return online, nil
}

func (self *BrokerConn) Release() {
	self.Using = false
}

type BrokerPool struct {
	pool map[string][]*BrokerConn // like as {"server1":[client1,client2], "server2":[client1, client2]}

	maxConn int //max connect for each broker server
	timeout int
	lock    sync.Mutex //
}

func NewBrokerPool(maxConn int, timeout int) *BrokerPool {
	pool := &BrokerPool{}
	pool.maxConn = maxConn
	pool.timeout = timeout
	pool.pool = map[string][]*BrokerConn{}
	return pool
}

func GetAddrLength(addrs []string) map[string]int {
	resultMap := make(map[string]int)
	for _, val := range addrs {
		clients := []*BrokerConn{}
		clients = brokerPool.pool[val]
		resultMap[val] = len(clients)
	}
	return resultMap
}

func (self *BrokerPool) GetBrokerConn(addr string, trywait bool) (*BrokerConn, string) {

	var conn *BrokerConn = nil
	var errMsg string = ""

	begin := time.Now()

	for {
		self.lock.Lock()

		clients := []*BrokerConn{}
		if self.pool[addr] != nil {
			clients = self.pool[addr]
			index := 0
			length := len(clients)
			for {
				if index == length {
					break
				}
				client := clients[index]
				if !client.Using {
					if client.IsActive() {
						conn = client
						conn.Using = true
						break
					} else {
						client.conn.Close()
						clients = append(clients[:index], clients[(index+1):]...)
						length--
						continue
					}
				}
				index++
			}
		}

		dis := int(time.Now().Sub(begin).Seconds())
		if conn != nil {
			//get connection or timeout
			errMsg = "conn exist"
			self.lock.Unlock()
			break
		} else if self.timeout > 0 && dis > self.timeout {
			errMsg = fmt.Sprintf("conn time out <%d>", dis)
			self.lock.Unlock()
			break
		} else {
			if len(clients) < self.maxConn {
				conn = NewBrokerConn(addr, self.timeout)
				if conn != nil {
					errMsg = "conn new"
					clients = append(clients, conn)
					self.pool[addr] = clients
					conn.Using = true
					self.lock.Unlock()
					break
				}
			}

			self.lock.Unlock()
			//if trywait {
			//	time.Sleep(5 * time.Second)
			//	continue
			//} else {
			//	break
			//}
			break
		}
	}

	return conn, errMsg
}
