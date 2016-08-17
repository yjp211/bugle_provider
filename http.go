package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

func HttpPostJson(httpUrl string, headers map[string]string, params interface{}, timeout int) (Dict, Error) {
	log.Debug("http post <%v> to %s", params, httpUrl)

	jstr, _ := json.Marshal(params)
	body := bytes.NewBuffer(jstr)
	log.Debug("http post <%v> to %s", string(jstr), httpUrl)

	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, time.Second*time.Duration(timeout)) //设置建立连接超时
				if err != nil {
					return nil, err
				}
				c.SetDeadline(time.Now().Add(time.Second * time.Duration(timeout))) //设置发送接收数据超时
				return c, nil
			},
		},
	}
	req, _ := http.NewRequest("POST", httpUrl, body)
	req.Header.Set("Content-Type", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error("post<%v> to <%s>failed, %s", params, httpUrl, err)
		return nil, NewError(REMOTE_CONN_ERR, err, "Remote serever can't access")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Error("post<%s> failed,err code: %d", httpUrl, resp.StatusCode)
		return nil, NewError(REMOTE_RESP_ERR, err, fmt.Sprintf("err code: %d", resp.StatusCode))
	}

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("post<%s> failed, %s", httpUrl, err)
		return nil, NewError(REMOTE_RESP_ERR, err, "Remote serever response error")
	}

	dict := Dict{}
	json.Unmarshal([]byte(result), &dict)

	log.Debug("http response:<%v>", dict)

	return dict, OK
}

func TransProviderResult(dict Dict) (Dict, Error) {
	code64, ok := dict["err_code"].(float64)
	if !ok {
		return nil, NewError(REMOTE_RESP_ERR, nil, "invalid data format")
	}
	code := int(code64)
	if code == 200 {
		data, ok := dict["data"].(map[string]interface{})
		if !ok {
			return nil, OK
		}
		return data, OK
	} else {
		if msg, ok := dict["err_msg"].(string); ok {
			return nil, NewError(code, nil, msg)
		} else {
			return nil, NewError(code, nil, "invalid data format")
		}
	}
}

func HttpGetJson(httpUrl string, params Dict, timeout int) (Dict, Error) {
	//fmt.Printf("http get<%v> to %s\n", params, httpUrl)

	l := httpUrl
	if params != nil {
		buf := ""
		for k, v := range params {
			buf = fmt.Sprintf("%s%s=%s&", buf, k, v)
		}
		if len(buf) > 0 {
			buf = buf[0 : len(buf)-1]
		}
		l = fmt.Sprintf("%s?%s", httpUrl, buf)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				c, err := net.DialTimeout(netw, addr, time.Second*time.Duration(timeout)) //设置建立连接超时
				if err != nil {
					return nil, err
				}
				c.SetDeadline(time.Now().Add(time.Second * time.Duration(timeout))) //设置发送接收数据超时
				return c, nil
			},
		},
	}
	req, _ := http.NewRequest("GET", l, nil)

	resp, err := client.Do(req)
	if err != nil {
		log.Error("get<%v> to <%s>failed, %s", params, httpUrl, err)
		return nil, NewError(REMOTE_CONN_ERR, err, "Remote serever can't access")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Error("get<%s> failed,err code: %d", httpUrl, resp.StatusCode)
		return nil, NewError(REMOTE_RESP_ERR, err, fmt.Sprintf("err code: %d", resp.StatusCode))
	}

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("get<%s> failed, %s", httpUrl, err)
		return nil, NewError(REMOTE_RESP_ERR, err, "Remote serever response error")
	}
	//fmt.Printf("http response:<%s>\n", string(result))

	dict := Dict{}
	json.Unmarshal([]byte(result), &dict)

	log.Debug("http response:<%v>", dict)

	return dict, OK
}
