package main

type OnlineForm struct {
	Topic string
}

type TokenForm struct {
	Device string
	Mac    string
	Ip     string

	Version int `json:"-"`
}

type PublishForm struct {
	UpstreamId string
	Topic      string
	Bridge     bool
	Msg        string
	Weight     int //消息权重
	Online     int64
	Data       string `json:"-"`
	Ttl        int    `json:"-"` //生命周期
	Version    int    `json:"-"`
	PubTime    int64  `json:"-"`
	Invoker    string `json:"-"`
}

type PublishData struct {
	PublishId string `json:"id"`
	Online    int64  `json:"online"`
	Total     int    `json:"total"`
	Datas     string `json:"datas"`
}
