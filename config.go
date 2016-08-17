package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)

type Config struct {
	//启动pprof
	enableOnlinePprof bool
	//后台访问口令
	backendPasswd string

	listenPort int
	logPath    string
	logLevel   string

	publishMaxWeight int
	//每秒最多接收的消息条数
	publishMaxCount int64

	//处理消息的并发数
	publishMaxMulti int

	//每条消息广播的处理能力
	publishMaxQps int64

	totalOnlineCacheExpire int
	localOnlineCacheExpire int

	httpRpcTimeout int

	relayList    []string
	relayInvoker string

	bridgeList    []string
	bridgeInvoker string

	requestInvokerKey string
	requestSignKey    string

	urlOnline  string
	urlToken   string
	urlPublish string

	urlCollectOnline string
	urlRelayPublish  string
	urlBridgePublish string

	invokerMap  map[string]interface{}
	decorateMap map[string]interface{}

	/* client config*/
	clientPingInterval      int //心跳间隔
	clientPingFailedCount   int //心跳失败多少次进行重连
	clientReconnectInterval int //重连间隔

	/* broker config*/
	brokerProxyAddr string //接入层负载均衡地址
	brokerProxyPort int
	brokerAddrs     []string //接入层内网地址列表

	brokerPoolMax int
	brokerTimeout int
}

func ParseConfig(configPath string, config *Config) error {
	contents, err := ioutil.ReadFile(configPath)
	if nil != err {
		fmt.Println("read config file %s error, %v", configPath, err)
		return err
	}

	dict := Dict{}
	err = json.Unmarshal(contents, &dict)
	if nil != err {
		fmt.Println("parse config file %s error, %v", configPath, err)
		return err
	}

	providerDict := dict["Provider"]
	if nil == providerDict {
		fmt.Println("config file %s:%s format  error", configPath, dict)
		return err
	}

	for key, val := range providerDict.(map[string]interface{}) {
		switch key {
		case "EnableOnlinePprof":
			config.enableOnlinePprof = val.(bool)
		case "BackendPasswd":
			config.backendPasswd = val.(string)

		case "ListenPort":
			config.listenPort = int(val.(float64))
		case "LogPath":
			config.logPath = val.(string)
		case "LogLevel":
			config.logLevel = val.(string)

		case "PublishMaxWeight":
			config.publishMaxWeight = int(val.(float64))
		case "PublishMaxCount":
			config.publishMaxCount = int64(val.(float64))
		case "PublishMaxMulti":
			config.publishMaxMulti = int(val.(float64))
		case "PublishMaxQps":
			config.publishMaxQps = int64(val.(float64))

		case "TotalOnlineCacheExpire":
			config.totalOnlineCacheExpire = int(val.(float64))
		case "LocalOnlineCacheExpire":
			config.localOnlineCacheExpire = int(val.(float64))

		case "HttpRpcTimeout":
			config.httpRpcTimeout = int(val.(float64))

		case "RelayList":
			str := val.(string)
			strArr := strings.Split(str, ",")
			arr := []string{}
			for _, v := range strArr {
				nv := strings.Trim(v, " ")
				if len(nv) > 0 {
					arr = append(arr, nv)
				}
			}
			config.relayList = arr
		case "RelayInvoker":
			config.relayInvoker = val.(string)

		case "BridgeList":
			str := val.(string)
			strArr := strings.Split(str, ",")
			arr := []string{}
			for _, v := range strArr {
				nv := strings.Trim(v, " ")
				if len(nv) > 0 {
					arr = append(arr, nv)
				}
			}
			config.bridgeList = arr
		case "BridgeInvoker":
			config.bridgeInvoker = val.(string)

		case "RequestInvokerKey":
			config.requestInvokerKey = val.(string)
		case "RequestSignKey":
			config.requestSignKey = val.(string)

		case "UrlOnline":
			config.urlOnline = val.(string)
		case "UrlToken":
			config.urlToken = val.(string)
		case "UrlPublish":
			config.urlPublish = val.(string)

		case "UrlCollectOnline":
			config.urlCollectOnline = val.(string)
		case "UrlRelayPublish":
			config.urlRelayPublish = val.(string)
		case "UrlBridgePublish":
			config.urlBridgePublish = val.(string)

		}

	}

	invokerDict := dict["Provider-Invoker"]
	if nil == invokerDict {
		fmt.Println("config file %s:%s format  error", configPath, dict)
		return err
	}
	config.invokerMap = invokerDict.(map[string]interface{})

	config.decorateMap = dict["Provider-Online-Decorate"].(map[string]interface{})
	if nil == config.decorateMap {
		fmt.Println("config file %s:%s format  error", configPath, dict)
		return err
	}

	clientDict := dict["Client"]
	if nil == clientDict {
		fmt.Println("config file %s:%s format  error", configPath, dict)
		return err
	}

	for key, val := range clientDict.(map[string]interface{}) {
		switch key {
		case "PingInterval":
			config.clientPingInterval = int(val.(float64))
		case "PingFailedCount":
			config.clientPingFailedCount = int(val.(float64))
		case "ReconnectInterval":
			config.clientReconnectInterval = int(val.(float64))
		}
	}

	brokerDict := dict["Broker"]
	if nil == brokerDict {
		fmt.Println("config file %s:%s format  error", configPath, dict)
		return err
	}

	for key, val := range brokerDict.(map[string]interface{}) {
		switch key {
		case "ProxyAddr":
			config.brokerProxyAddr = val.(string)
		case "ProxyPort":
			config.brokerProxyPort = int(val.(float64))
		case "BrokerAddrs":
			str := val.(string)
			strArr := strings.Split(str, ",")
			arr := []string{}
			for _, v := range strArr {
				nv := strings.Trim(v, " ")
				if len(nv) > 0 {
					arr = append(arr, nv)
				}
			}
			config.brokerAddrs = arr

		case "PoolMaxConn":
			config.brokerPoolMax = int(val.(float64))
		case "PoolTimeout":
			config.brokerTimeout = int(val.(float64))
		}
	}

	return OK

}
