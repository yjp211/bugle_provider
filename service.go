package main

import (
	"fmt"
	"sort"
)

const (
	Guest_Account = "Guest"
	Guest_Passwd  = "Guest"
)

func gainHttpUrl(server string, port int, sufix string) string {
	return fmt.Sprintf("http://%s:%d%s", server, port, sufix)
}

/**
获取token
*/
func ServiceGetToken(form *TokenForm) (Dict, Error) {

	data := Dict{}

	data["account"] = Guest_Account
	data["password"] = Guest_Passwd

	data["pingInterval"] = configOpt.clientPingInterval
	data["pingFailedCount"] = configOpt.clientPingFailedCount
	data["reconnectInterval"] = configOpt.clientReconnectInterval

	data["brokerAddr"] = configOpt.brokerProxyAddr
	data["brokerPort"] = configOpt.brokerProxyPort

	return data, OK
}

/**
*分布式部署在线人数要分开统计， 这是一个对内接口，返回本中心的在线数据
 */
func ServiceGetLocalOnline(topic string) (Dict, Error) {
	//返回本地在线人数不要去修饰
	online := onlineCache.GetLocalOnline(topic)

	data := Dict{
		"online": online,
	}
	return data, OK
}

/**
*分布式部署在线人数要分开统计， 这是一个对外接口，返回所有中心的在线数据
 */
func ServiceGetOnline(topic string) (Dict, Error) {
	var online int64 = 0

	decorate := GetDecorate(topic)
	if decorate >= 0 {
		online = int64(decorate * float64(onlineCache.GetTotalOnline(topic)))
	} else { //表示利用一个固定值 作为在线人数，没必须去后台捞数据，减少穿透
		online = 0 - int64(decorate)
	}

	data := Dict{
		"online": online,
	}
	return data, OK
}

//获取在线人数，对外后台接口，不加权在线人数
func ServiceGetPureOnline(topic string) (Dict, Error) {
	online := onlineCache.GetTotalOnline(topic)
	data := Dict{
		"online": online,
	}

	return data, OK
}

type Pair struct {
	Key   string
	Value int64
}

// A slice of pairs that implements sort.Interface to sort by values
type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }

//获取所有在线人数，对外后台接口，不加权在线人数
func ServiceGetAllPureOnline(showlen, local int) (Dict, Error) {
	curMap := onlineCache.GetAllTotalOnline(local == 1)
	maplen := len(curMap)
	pairList := make(PairList, maplen)
	i := 0
	total := int64(0)
	for k, v := range curMap {
		pairList[i] = Pair{k, v}
		i++
		total += v
	}
	sort.Sort(sort.Reverse(pairList))
	if showlen <= 0 || showlen >= maplen {
		showlen = maplen
	}
	data := Dict{
		"1-total-online": total,
		"2-topic-count":  maplen,
		"3-show-count":   showlen,
		"4-topic-tail":   pairList[0:showlen],
	}

	return data, OK
}

//来自集群内的广播，直接将消息广播到broker
func ServiceRelayPublish(form *PublishForm) Error {
	form.PubTime = Gtimer.Unix
	return PushToLocal(form)
}

//集群间广播
func ServiceBridgePublish(form *PublishForm) Error {

	//桥接过来的消息要重新进行 过载保护处理
	if !InrcCurCountAndTryTrans() {
		log.Error("消息<%s> 超出处理能力，被丢弃", form.UpstreamId)
		return OK
	}

	form.PubTime = Gtimer.Unix

	//重新计算在线人数，(bridge过来的在线人数是其它集群的)
	//在线人数只是本集群之内的数据
	//获取在线人数 修饰手法
	decorate := GetDecorate(form.Topic)
	if decorate >= 0 {
		form.Online = int64(decorate * float64(onlineCache.GetTotalOnline(form.Topic)))
	} else { //表示利用一个固定值 作为在线人数，没必须去后台捞数据，减少穿透
		form.Online = 0 - int64(decorate)
	}

	//转到本节点broker
	PushToLocal(form)
	//转到集群内其它节点
	return RelayInCluster(form)
}

//对外接口
func ServicePublish(form *PublishForm) Error {

	//压力过载保护
	//本集群处理不过来的消息，不会进行任何处理, 不桥接、不转发
	if !InrcCurCountAndTryTrans() {
		log.Error("消息<%s> 超出处理能力，被丢弃", form.UpstreamId)
		return OK
	}

	if len(form.UpstreamId) == 0 {
		form.UpstreamId = NewUuid(true)
	}

	//获取在线人数 修饰手法
	decorate := GetDecorate(form.Topic)
	if decorate >= 0 {
		form.Online = int64(decorate * float64(onlineCache.GetTotalOnline(form.Topic)))
	} else { //表示利用一个固定值 作为在线人数，没必须去后台捞数据，减少穿透
		form.Online = 0 - int64(decorate)
	}

	//先进入本地过滤系统
	PushToLocal(form)

	//广播到集群内的其它节点
	RelayInCluster(form)

	if form.Bridge {
		//桥接消息到其它集群
		BridgeBetweenCluster(form)
	}

	return OK
}
