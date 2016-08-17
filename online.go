package main

/**
 * 在线人数统计方式
 * 1、建立一个全局缓存map: 缓存topic在线人数
 * 2、每隔3秒标记一次缓存失效（不删除原数据，如果下次获取出错，需要继续使用现有数据）
 * 3、获取topic在线人数先命中缓存，缓存没有则进行分布式拉取
 * 4、向各分布式代理节点发送统计命令，如果有一个出错超时，则沿用原缓存
 * 4、定时器每秒进行重置
 */

//如果没有
//要去抢夺资源锁，防止并发

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	onlineCache *OnlineCache
)

//子主题 有效客户端缓存一秒
//父主题 有效客户端缓存3秒
type OnlineCache struct {
	TotalOnline map[string]int64
	TotalExpire map[string]bool

	LocalOnline map[string]int64
	LocalExpire map[string]bool
}

func CreateOnlineCache() {
	onlineCache = &OnlineCache{
		TotalOnline: map[string]int64{},
		TotalExpire: map[string]bool{},
		LocalOnline: map[string]int64{},
		LocalExpire: map[string]bool{},
	}

	onlineCache.startWatch()
}

func (p *OnlineCache) expireTotalCache() {
	//定时标记缓存不可用
	for k := range p.TotalExpire {
		p.TotalExpire[k] = true
	}
}

func (p *OnlineCache) expireLocalCache() {
	//定时标记缓存不可用
	for k := range p.LocalExpire {
		p.LocalExpire[k] = true
	}
}

func (p *OnlineCache) cleanCache() {
	//定时标记缓存不可用
	for k := range p.TotalExpire {
		delete(p.TotalOnline, k)
		delete(p.TotalExpire, k)
	}

	for k := range p.LocalExpire {
		delete(p.LocalOnline, k)
		delete(p.LocalExpire, k)
	}

}

func (p *OnlineCache) startWatch() {
	go func() { //定时标记缓存不可用
		for {
			//log.Debug("sub topic push clients expire")
			p.expireTotalCache()

			<-time.After(time.Second *
				time.Duration(configOpt.totalOnlineCacheExpire))
		}
	}()

	go func() { //定时标记缓存不可用
		for {
			//log.Debug("sub topic push clients expire")
			p.expireLocalCache()

			<-time.After(time.Second *
				time.Duration(configOpt.localOnlineCacheExpire))
		}
	}()

	go func() { //每隔3小时清空所有缓存，防止堆积垃圾数据
		for {
			//log.Debug("sub topic push clients expire")
			p.cleanCache()

			<-time.After(time.Hour * 3)
		}
	}()

}

func (p *OnlineCache) updateLocal(topic string) int64 {
	//获取最新记录
	online, _ := CollectLocalOnline(topic)

	p.LocalOnline[topic] = online
	p.LocalExpire[topic] = false
	return online
}

func (p *OnlineCache) GetLocalOnline(topic string) int64 {
	online, ok := p.LocalOnline[topic]
	if ok { //已经存在记录，直接返回
		if expire, _ := p.LocalExpire[topic]; expire { //已经过期了，异步更新下数据，方便下次的人用
			go p.updateLocal(topic)

		}
		return online
	}
	//不存在记录，则必须等待返回
	online = p.updateLocal(topic)
	return online
}

func (p *OnlineCache) updateTotal(topic string) int64 {
	//获取最新记录
	online, _ := CollectTotalOnline(topic)

	p.TotalOnline[topic] = online
	p.TotalExpire[topic] = false
	return online
}

func (p *OnlineCache) GetTotalOnline(topic string) int64 {
	online, ok := p.TotalOnline[topic]
	if ok { //已经存在记录，直接返回
		if expire, _ := p.TotalExpire[topic]; expire { //已经过期了，异步更新下数据，方便下次的人用
			go p.updateTotal(topic)

		}
		return online
	}
	//不存在记录，则必须等待返回
	online = p.updateTotal(topic)
	return online
}

func (p *OnlineCache) GetAllTotalOnline(local bool) map[string]int64 {
	if local {
		return p.LocalOnline
	} else {
		return p.TotalOnline
	}
}

/**
*分布式部署在线人数要分开统计， 这是一个对内接口，返回本中心的在线数据
 */
func CollectLocalOnline(topic string) (int64, bool) {

	var total int64 = 0
	//分散收集收集本中心和其它中心的数据
	var wg sync.WaitGroup
	wg.Add(len(configOpt.brokerAddrs))

	haveError := false

	//收集broker
	for _, addrStr := range configOpt.brokerAddrs {
		go func(addr string) {
			defer wg.Done()
			conn, _ := brokerPool.GetBrokerConn(addr, true)
			if nil == conn {
				log.Error("get connection to <%s> failed", addr)
				return
			}
			defer conn.Release()

			online, err := conn.QueryTopicOnline(topic)
			if nil != err {
				log.Error("query topic <%s> online count at local broker<%s>failed, %v",
					topic, addr, err)
				haveError = true
			} else {
				log.Debug("query topic <%s> online count at local broker<%s>success, %d",
					topic, addr, online)
			}
			atomic.AddInt64(&total, online)
		}(addrStr)
	}

	wg.Wait()

	return total, haveError
}

func CollectTotalOnline(topic string) (int64, bool) {
	var total int64 = 0
	//分散收集收集本中心和其它中心的数据
	var wg sync.WaitGroup
	wg.Add(len(configOpt.relayList) + 1)

	haveError := false

	form := &OnlineForm{
		Topic: topic,
	}

	//收集本地
	go func() {
		defer wg.Done()
		online := onlineCache.GetLocalOnline(topic)
		atomic.AddInt64(&total, online)
	}()

	//收集异地
	for _, addrStr := range configOpt.relayList {
		go func(addr string) {
			defer wg.Done()
			httpUrl := fmt.Sprintf("http://%s%s", addr, configOpt.urlCollectOnline)
			data, ret := HttpPostJson(httpUrl, nil, form, configOpt.httpRpcTimeout)
			if !ret.Ok() {
				log.Error("collect online for<%s> at <%s> failed, %s", topic, addr, ret)
				haveError = true
				return
			}
			data, ret = TransProviderResult(data)
			if !ret.Ok() {
				log.Error("collect online for<%s> at <%s> failed, %s", topic, addr, ret)
				haveError = true
				return
			}
			online, ok := data["online"].(float64)
			if !ok {
				log.Error("collect online for<%s> at <%s> failed, invalid data", topic, addr)
				haveError = true
				return
			}
			atomic.AddInt64(&total, int64(online))
		}(addrStr)
	}
	wg.Wait()

	return total, haveError
}
