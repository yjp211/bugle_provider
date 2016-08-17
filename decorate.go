package main

var (
	DecorateMap        map[string]float64 = nil
	DefaultDecorateKey                    = "default"
)

//大于等于0表示实际值 x 加权值
//小于0 表示用该值的绝对值 替换实际值
//大于1且是小数形式 2.xx  则对xx后面的尾数进行随机

func InitOnlineDecorteMap(dmap map[string]interface{}) {
	DecorateMap = map[string]float64{}
	for k, v := range dmap {
		val := v.(float64)
		DecorateMap[k] = val
	}
	//fmt.Printf("%+v\n", DecorateMap)
}

func PutDecorateMap(k string, v float64) {
	DecorateMap[k] = v
}

func GetDecorateMap(k string) float64 {
	v, _ := DecorateMap[k]
	return v
}

func GetDecorate(topic string) float64 {
	//先查找是否用具体的直播间加权规则
	v, ok := DecorateMap[topic]
	if ok {
		return v
	}

	//查找默认规则
	v, ok = DecorateMap[DefaultDecorateKey]
	if ok {
		return v
	}

	//一个都没找到返回0
	return 1
}
