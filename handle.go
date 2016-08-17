package main

import (
	"io/ioutil"
	"strconv"
	"strings"

	"encoding/json"
	"fmt"

	"github.com/yjp211/web"
)

func getPerAddress(ctx *web.Context) string {
	req := ctx.Request

	// We suppose RemoteAddr is of the form Ip:Port as specified in the Request
	// documentation at http://golang.org/pkg/net/http/#Request
	pos := strings.LastIndex(req.RemoteAddr, ":")
	if pos > 0 {
		return req.RemoteAddr[0:pos]
	} else {
		return req.RemoteAddr
	}
}

func jsonpWrap(ctx *web.Context, origin string) string {
	callback := ctx.Params["callback"]
	if len(callback) == 0 {
		return origin
	} else {
		return fmt.Sprintf("%s(%s)", callback, origin)
	}
}

func DoneToken(ctx *web.Context) string {
	log.Debug("--->get token ")

	ctx.SetHeader("Content-Type", "application/json; charset=UTF-8", true)

	var form *TokenForm
	params := map[string]string{}
	if ctx.Request.Method == "POST" {
		form = &TokenForm{}
		body, err := ioutil.ReadAll(ctx.Request.Body)
		if err != nil {
			return jsonpWrap(ctx, NewError(INVALID_PARAM, nil, "invalid params").Json())
		}
		err = json.Unmarshal(body, form)
		if err != nil {
			return jsonpWrap(ctx, NewError(INVALID_PARAM, nil, "invalid params").Json())
		}
	} else {
		params = ctx.Params
		form = &TokenForm{
			Device: params["device"],
			Mac:    params["mac"],
			Ip:     params["ip"],
		}
	}
	form.Version = 1
	form.Ip = getPerAddress(ctx)

	log.Debug("get token:<%v>", form)

	data, ret := ServiceGetToken(form)

	if !ret.Ok() {
		log.Error("<%+v>get token failed, %s", *form, ret)
	} else {
		log.Info("<%+v>get token success, %+v", *form, data)
		ret.Data = data
	}
	return jsonpWrap(ctx, ret.Json())

}

/**
*分布式部署在线人数要分开统计， 这是一个对内接口，返回本中心的在线数据
 */
func DoneCollectLocalOnline(ctx *web.Context) string {
	log.Debug("--->collect local online count")

	ctx.SetHeader("Content-Type", "application/json; charset=UTF-8", true)

	form := &OnlineForm{}
	body, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		return jsonpWrap(ctx, NewError(INVALID_PARAM, nil, "invalid params").Json())
	}
	err = json.Unmarshal(body, form)
	if err != nil {
		return jsonpWrap(ctx, NewError(INVALID_PARAM, nil, "invalid params").Json())
	}

	topic := form.Topic
	data, ret := ServiceGetLocalOnline(topic)

	if !ret.Ok() {
		log.Error("collect topic<%s> local online count failed, %s", topic, ret)
	} else {
		log.Info("collect topic<%s> local online count success, %+v", topic, data)
		ret.Data = data
	}
	return jsonpWrap(ctx, ret.Json())
}

/**
*分布式部署在线人数要分开统计， 这是一个对外接口，返回所有中心的在线数据
 */
func DoneGetOnline(ctx *web.Context) string {
	log.Debug("--->get online count")

	ctx.SetHeader("Content-Type", "application/json; charset=UTF-8", true)

	topic := ctx.Params["topic"]

	data, ret := ServiceGetOnline(topic)
	if !ret.Ok() {
		log.Error("get topic<%s> online count failed, %s", topic, ret)

	} else {
		log.Info("get topic<%s> online count success, %+v", topic, data)
		ret.Data = data
	}
	return jsonpWrap(ctx, ret.Json())

}

func gainPublishForm(ctx *web.Context) (*PublishForm, Error) {
	invoker := ctx.Request.Header.Get(configOpt.requestInvokerKey)
	sig := ctx.Request.Header.Get(configOpt.requestSignKey)
	if len(invoker) == 0 || len(sig) == 0 {
		log.Error("invalid request header")
		return nil, NewError(INVALID_PARAM, nil, "invalid request header")
	}

	dict, ok := configOpt.invokerMap[invoker].(map[string]interface{})
	if !ok {
		log.Error("invalid invoker:%s", invoker)
		return nil, NewError(INVALID_PARAM, nil, "invalid voker")
	}
	key, ok := dict["key"].(string)
	if !ok {
		log.Error("invalid invoker:%s", invoker)
		return nil, NewError(INVALID_PARAM, nil, "invalid voker")
	}

	form := &PublishForm{}
	body, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		log.Error("read params body faild, %s", err)
		return nil, NewError(INVALID_PARAM, nil, "invalid params")
	}
	err = json.Unmarshal(body, form)
	if err != nil {
		log.Error("parse params to json faild, %s", err)
		return nil, NewError(INVALID_PARAM, nil, "invalid params format")
	}
	rightSig := Md5Sig(string(body), invoker, key)
	if strings.ToLower(sig) != strings.ToLower(rightSig) {
		log.Error("invalid sig<%s> right is <%s>", sig, rightSig)
		return nil, NewError(INVALID_PARAM, nil, "invalid sign")
	}
	form.Invoker = invoker

	if form.Weight > configOpt.publishMaxWeight {
		form.Weight = configOpt.publishMaxWeight
	} else if form.Weight < 1 {
		form.Weight = 1
	}
	form.Ttl = form.Weight

	form.Version = 1
	return form, OK
}

//来自集群内广播的消息(集群内广播)
func DoneRelayPublish(ctx *web.Context) string {
	log.Debug("---->Relay Publish")
	ctx.SetHeader("Content-Type", "application/json; charset=UTF-8", true)

	form, ret := gainPublishForm(ctx)
	if !ret.Ok() {
		return ret.Json()
	}

	ret = ServiceRelayPublish(form)

	if !ret.Ok() {
		log.Error("relay publish<%+v> failed, %s", form, ret)

	} else {
		log.Info("relay publish<%+v> success", form)
	}

	return ret.Json()
}

//来自其它集群广播的消息(集群间广播)
func DoneBridgePublish(ctx *web.Context) string {
	log.Debug("---->Bridge Publish")
	ctx.SetHeader("Content-Type", "application/json; charset=UTF-8", true)

	form, ret := gainPublishForm(ctx)
	if !ret.Ok() {
		return ret.Json()
	}

	ret = ServiceBridgePublish(form)

	if !ret.Ok() {
		log.Error("bridge publish<%+v> failed, %s", form, ret)

	} else {
		log.Info("bridge publish<%+v> success", form)
	}

	return ret.Json()
}

//来自用户消息 (对外接口)
func DonePublish(ctx *web.Context) string {
	log.Debug("---->Publish")

	ctx.SetHeader("Content-Type", "application/json; charset=UTF-8", true)

	form, ret := gainPublishForm(ctx)
	if !ret.Ok() {
		return ret.Json()
	}

	ret = ServicePublish(form)

	if !ret.Ok() {
		log.Error("publish<%+v> failed, %s", form, ret)
	} else {
		log.Info("publish<%+v> success", form)
	}
	return ret.Json()
}

func haveBackendParam(ctx *web.Context) bool {
	backend, ok := ctx.Params["backend"]
	if !ok || len(backend) == 0 {
		return false
	}
	return backend == configOpt.backendPasswd
}

func DoneGetDecorate(ctx *web.Context) string {
	log.Debug("---->get decorate")
	ctx.SetHeader("Content-Type", "application/json; charset=UTF-8", true)

	if !haveBackendParam(ctx) {
		return NewError(NO_PERM, nil, "no perm").Json()
	}

	ret := OK
	ret.Data = Dict{
		"map": DecorateMap,
	}
	return ret.Json()
}

func DoneSetDecorate(ctx *web.Context) string {
	log.Debug("---->set decorate")
	ctx.SetHeader("Content-Type", "application/json; charset=UTF-8", true)

	if !haveBackendParam(ctx) {
		return NewError(NO_PERM, nil, "no perm").Json()
	}

	key, ok := ctx.Params["key"]
	if !ok {
		return NewError(INVALID_PARAM, nil, "invalid params").Json()
	}

	val, err := strconv.ParseFloat(ctx.Params["val"], 64)
	if err != nil {
		return NewError(INVALID_PARAM, nil, "invalid params").Json()
	}

	PutDecorateMap(key, val)
	ret := OK
	ret.Data = Dict{
		"map": DecorateMap,
	}
	return ret.Json()
}

//获取没有加权的在线人数
func DoneGetPureOnline(ctx *web.Context) string {
	log.Debug("--->get pure online count")

	ctx.SetHeader("Content-Type", "application/json; charset=UTF-8", true)

	if !haveBackendParam(ctx) {
		return NewError(NO_PERM, nil, "no perm").Json()
	}

	topic := ctx.Params["topic"]

	data, ret := ServiceGetPureOnline(topic)

	if !ret.Ok() {
		log.Error("get topic<%s> online count failed, %s", topic, ret)

	} else {
		log.Info("get topic<%s> online count success, %+v", topic, data)
		ret.Data = data
	}
	return ret.Json()
}

//获取所有的在线人数
func DoneGetAllPureOnline(ctx *web.Context) string {
	log.Debug("--->get all  pure online count")

	ctx.SetHeader("Content-Type", "application/json; charset=UTF-8", true)

	if !haveBackendParam(ctx) {
		return NewError(NO_PERM, nil, "no perm").Json()
	}
	showlen, err := strconv.Atoi(ctx.Params["len"])
	if err != nil {
		showlen = 0
	}

	local, err := strconv.Atoi(ctx.Params["local"])
	if err != nil {
		local = 0
	}

	data, ret := ServiceGetAllPureOnline(showlen, local)

	if !ret.Ok() {
		log.Error("get all online count failed, %s", ret)
	} else {
		log.Info("get all  online count success, %+v", data)
		ret.Data = data
	}
	return ret.Json()
}
