package main

import "encoding/json"

const (
	SUCCESS = iota

	REMOTE_CONN_ERR
	REMOTE_RESP_ERR

	BROKER_CONN_ERR
	BROKER_PUSH_ERR

	NO_PERM          = 401
	INVALID_PARAM    = 400
	SYSTEM_BUSY      = 503
	SERVICE_DEGRADED = 501 //服务降级
)

type Error struct {
	Code int   // error code
	Err  error // origin error
	Msg  string
	Data Dict
}

func (e Error) String() string {
	if e.Err != nil {
		return e.Msg + ", " + e.Err.Error()
	}
	return e.Msg
}

func (e Error) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Msg
}

func (e Error) Ok() bool {
	if e.Code == SUCCESS {
		return true
	}
	return false
}

func (e Error) Json() string {
	dict := Dict{}
	if e.Code == SUCCESS {
		dict["err_code"] = 200
		dict["err_msg"] = "操作成功"
	} else {
		dict["err_code"] = e.Code
		dict["err_msg"] = e.Msg
	}
	dict["data"] = e.Data
	jstr, _ := json.Marshal(dict)
	return string(jstr)
}

var OK Error = Error{SUCCESS, nil, "", nil}

func NewError(code int, err error, msg string) Error {
	return Error{code, err, msg, nil}
}
