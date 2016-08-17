package main

/**
 * 推送平绿的控制算法
 * 1、将当前的推送按照不同的权重插到不同的队列中
 * 2、每插入一次，通知消费协程一次
 * 3、消息协程每次都从高到低，一次遍历所有队列
 * 4、找到一条消息则处理，处理完后等待下次循环再次从头开始
 * 5、消息协程推送到共享层，如果共享控制层的流控返回繁忙
 * 6、繁忙，则重新将此消息加入到推送队列
 * 7、消息具有生命周期ttl，当周期已结束消息不再会进入循环
 */
import (
	"encoding/json"
	"sync"
	"sync/atomic"
)

var (
	WeightQueueMap   = map[int]*Queue{}
	MaxPublishWeight = configOpt.publishMaxWeight
	NewJob           = make(chan bool, configOpt.publishMaxCount)

	//每条处理消息的条数
	MaxPublishCount int64 = configOpt.publishMaxCount
	CurSecondCount  int64 = 0
	CountExceed           = false

	//每秒推送消息的客户端数
	MaxPublishQps int64 = configOpt.publishMaxQps
	CurSecondQps  int64 = 0
	QpsExceed           = false
)

func ResetCurCount() {
	atomic.StoreInt64(&CurSecondCount, 0)
	CountExceed = false
}

func ResetCurQps() {
	atomic.StoreInt64(&CurSecondQps, 0)
	QpsExceed = false
}

func InrcCurCountAndTryTrans() bool {
	if CountExceed {
		return false
	}
	neew := atomic.AddInt64(&CurSecondCount, 1)
	if neew > MaxPublishCount {
		CountExceed = true
		return false
	}
	return true
}
func InrcCurQpsAndTryTrans(qps int64) (bool, int64) {
	if QpsExceed {
		return false, 0
	}
	neew := atomic.AddInt64(&CurSecondQps, qps)
	if neew > MaxPublishQps {
		QpsExceed = true
		return false, neew
	}
	return true, neew
}

func SetMaxPublishCount(count int64) {
	MaxPublishCount = count
	NewJob = make(chan bool, count)
}

func SetMaxPublishQps(qps int64) {
	MaxPublishQps = qps
}

func SetMaxPublishWeight(weight int) {
	MaxPublishWeight = weight
}

type Item struct {
	Data *PublishForm
	Next *Item
}

type Queue struct {
	//推送队列
	Length int64 //队列长度
	Head   *Item //队列头
	Tail   *Item //队列尾
	Lock   sync.Mutex
}

func (p *Queue) push(pub *PublishForm, reuse bool) {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	item := &Item{
		Data: pub,
		Next: nil,
	}

	//当前队列为空
	if p.Length == 0 {
		p.Head = item
		p.Tail = item
	} else {
		//队列不为空

		//新消息，追加到队列末尾
		if !reuse {
			p.Tail.Next = item
			p.Tail = item
		} else {
			//回收的消息，需要插到时间顺序的位置
			//插到最前
			if p.Head.Data.PubTime > item.Data.PubTime {
				item.Next = p.Head
				p.Head = item
			} else {
				cur := p.Head
				var point *Item = nil
				j := 0
				for cur != nil {
					//没有到末尾
					j++
					if cur.Next != nil {
						//当前的下一个节点发布时间比这个节点要晚
						if cur.Next.Data.PubTime > item.Data.PubTime {
							point = cur
							break
						}
					}
					cur = cur.Next
				}
				//后插（因为查找的是前一个节点）
				if cur != nil {
					tmp := cur.Next
					point.Next = item
					item.Next = tmp
				} else {
					//插到末尾
					p.Tail.Next = item
					p.Tail = item
				}
			}

		}

	}
	//队列长度+1
	p.Length++
}

func (p *Queue) pop() *PublishForm {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	if p.Length == 0 {
		return nil
	}

	cur := p.Head
	p.Head = cur.Next
	pub := cur.Data
	p.Length -= 1
	if p.Length == 0 {
		p.Tail = nil
	}

	return pub

}

func CollectPublish(pub *PublishForm, reuse bool) {
	weight := pub.Weight
	queue, ok := WeightQueueMap[weight]
	if !ok {
		//消息权重不在指定范围，丢弃
		//not support ignore
		return
	}
	queue.push(pub, reuse)

	log.Debug("---deliver new job")
	go func() {
		NewJob <- true
	}()
}

func StartPublishConsumerLoop(multi int) {
	/**先初始化所有的queue*/
	for i := 0; i <= MaxPublishWeight; i++ {
		queue := &Queue{
			Length: 0,
			Head:   nil,
			Tail:   nil,
			Lock:   sync.Mutex{},
		}
		WeightQueueMap[i] = queue
	}

	for i := 0; i < multi; i++ {
		go ConsumerPublish()
	}
}

func ConsumerPublish() {
	for {
		<-NewJob

		for i := MaxPublishWeight; i >= 0; i-- {
			queue, ok := WeightQueueMap[i]
			if !ok {
				continue
			}
			pub := queue.pop()
			if pub == nil {
				continue
			}
			if pub.PubTime+int64(pub.Ttl) < Gtimer.Unix {
				continue
			}
			log.Debug("---->publish catch, weight: %d, id:%s", pub.Weight, pub.UpstreamId)

			ret := spreadToBrokers(pub)
			if !ret.Ok() {
				//如果是系统繁忙没有进行推送，则将此消息塞回到推送队列中去
				if ret.Code == SYSTEM_BUSY {
					CollectPublish(pub, true)
				}
			}

		}

	}
}

func spreadToBrokers(pub *PublishForm) Error {
	//获取真正的本地在线用户数
	online := onlineCache.GetLocalOnline(pub.Topic)
	flag, onlineQps := InrcCurQpsAndTryTrans(online)
	if !flag {
		log.Error("system busy <%d>", onlineQps)
		return NewError(SYSTEM_BUSY, nil, "system busy")
	}

	//设置消息体
	data := &PublishData{
		PublishId: pub.UpstreamId,
		Total:     1, //目前只会有一条消息
		Online:    pub.Online,
		Datas:     pub.Msg,
	}

	jstr, _ := json.Marshal(data)
	pub.Data = string(jstr)
	for _, addrStr := range configOpt.brokerAddrs {
		go func(addr string) {
			conn, errMsg := brokerPool.GetBrokerConn(addr, true)
			if nil == conn {
				log.Error("get connection to <%s> failed, %s", addr, errMsg)
				return
			}
			defer conn.Release()
			err := conn.PublishPureMsg(pub.UpstreamId, pub.Topic, pub.Data)
			if nil != err {
				log.Error("publish message<%s> to local broker<%s>failed, %v",
					pub.UpstreamId, addr, err)
			} else {
				log.Debug("publish message<%s> to local broker<%s> success",
					pub.UpstreamId, addr)
			}
		}(addrStr)
	}

	return OK
}
