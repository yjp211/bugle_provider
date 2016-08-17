package main

import (
	"encoding/json"
	"fmt"
)

//将消息发到本节点的broker
func PushToLocal(pub *PublishForm) Error {
	pub.PubTime = Gtimer.Unix
	CollectPublish(pub, false)
	return OK
}

//集群内转发
func RelayInCluster(pub *PublishForm) Error {
	if len(configOpt.relayList) == 0 {
		return OK
	}
	return providerToRemote(pub, configOpt.relayInvoker,
		configOpt.urlRelayPublish, configOpt.relayList)
}

//集群间桥接
func BridgeBetweenCluster(pub *PublishForm) Error {
	if len(configOpt.bridgeList) == 0 {
		return OK
	}
	return providerToRemote(pub, configOpt.bridgeInvoker,
		configOpt.urlBridgePublish, configOpt.bridgeList)
}

//将消息发送到其它provider
func providerToRemote(pub *PublishForm,
	invoker string, url string, providerList []string) Error {
	jstr, _ := json.Marshal(pub)
	dict, ok := configOpt.invokerMap[invoker].(map[string]interface{})
	if !ok {
		log.Error("invalid invoker:%s", invoker)
		return NewError(INVALID_PARAM, nil, "invalid invoker")
	}
	key, ok := dict["key"].(string)
	if !ok {
		log.Error("invalid invoker:%s", invoker)
		return NewError(INVALID_PARAM, nil, "invalid invoker")
	}

	sig := Md5Sig(string(jstr), invoker, key)

	headerMap := map[string]string{
		configOpt.requestInvokerKey: invoker,
		configOpt.requestSignKey:    sig,
	}

	goProviderPublish(pub, headerMap, url, providerList)

	return OK
}

func goProviderPublish(pub *PublishForm, headerMap map[string]string,
	url string, providerList []string) {
	for _, addrStr := range providerList {
		go func(addr string) {
			httpUrl := fmt.Sprintf("http://%s%s", addr, url)
			data, ret := HttpPostJson(httpUrl, headerMap, pub, configOpt.httpRpcTimeout)
			if !ret.Ok() {
				log.Error("publish to provider:<%s> to <%s> failed, %s", pub.UpstreamId, addr, ret)
				return
			}
			_, ret = TransProviderResult(data)
			if !ret.Ok() {
				log.Error("publish to provider:<%s> to <%s> failed, %s", pub.UpstreamId, addr, ret)
			} else {
				log.Info("publish provider:<%s> to <%s> success", pub.UpstreamId, addr)
			}
		}(addrStr)

	}
}
