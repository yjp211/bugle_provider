package main

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/op/go-logging"
	"github.com/yjp211/web"
)

type Dict map[string]interface{}
type List []interface{}

var (
	CONFIG_PATH = flag.String("c", "./config.conf", "config path")
	CONFIG_PORT = flag.Int("p", 0, "config port")
	log         *logging.Logger
	Gtimer      *Timer
	brokerPool  *BrokerPool
	configOpt   Config = Config{}
)

func main() {
	flag.Parse()
	ParseConfig(*CONFIG_PATH, &configOpt)
	fmt.Printf("%+v\n", configOpt)

	runtime.GOMAXPROCS(runtime.NumCPU())

	log = logging.MustGetLogger("provider")

	logLevel, err := logging.LogLevel(configOpt.logLevel)
	if nil != err {
		logLevel = logging.DEBUG
	}
	InitLogger(configOpt.logPath, logLevel)

	SetMaxPublishCount(configOpt.publishMaxCount)
	SetMaxPublishQps(configOpt.publishMaxQps)
	SetMaxPublishWeight(configOpt.publishMaxWeight)

	//明确指定-p参数，使用-p, 否则仍然读配置文件
	if *CONFIG_PORT > 0 {
		configOpt.listenPort = *CONFIG_PORT
	}

	InitOnlineDecorteMap(configOpt.decorateMap)
	CreateOnlineCache()

	Gtimer = NewTimer()
	Gtimer.Start()
	StartPublishConsumerLoop(configOpt.publishMaxMulti)

	brokerPool = NewBrokerPool(configOpt.brokerPoolMax, configOpt.brokerTimeout)

	log.Debug("----------begin---------")
	web.Get(configOpt.urlToken, DoneToken)

	/*对外的主要3个接口*/
	web.Get(configOpt.urlOnline, DoneGetOnline)
	web.Get(configOpt.urlToken, DoneToken)
	web.Post(configOpt.urlPublish, DonePublish)

	/**聊天室内部转发相关接口*/
	web.Post(configOpt.urlCollectOnline, DoneCollectLocalOnline)
	web.Post(configOpt.urlRelayPublish, DoneRelayPublish)
	web.Post(configOpt.urlBridgePublish, DoneBridgePublish)

	/**后端控制接口*/
	web.Get("/provider/v1/backend/online", DoneGetPureOnline)
	web.Get("/provider/v1/backend/decorate", DoneGetDecorate)
	web.Post("/provider/v1/backend/decorate", DoneSetDecorate)

	web.Get("/provider/v1/backend/online/all", DoneGetAllPureOnline)

	listen := fmt.Sprintf("0.0.0.0:%d", configOpt.listenPort)
	web.Config.Profiler = configOpt.enableOnlinePprof
	web.Run(listen)
}
