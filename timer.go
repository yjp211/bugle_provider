package main

import (
	"time"
)

type Timer struct {
	Unix int64
}

func NewTimer() *Timer {
	timer := &Timer{}
	timer.Unix = time.Now().Unix()
	return timer
}

func (p *Timer) Start() {
	go func() {
		for {
			cur := <-time.After(time.Second)
			p.Unix = cur.Unix()

			ResetCurCount()
			ResetCurQps()

		}

	}()
}
