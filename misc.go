package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"strings"

	"github.com/satori/go.uuid"
)

func NewUuid(strip bool) string {
	u := uuid.NewV4().String()
	if strip {
		u = strings.Replace(u, "-", "", -1)
	}
	return u
}

func Md5Sig(body, invoker, key string) string {
	buf := fmt.Sprintf("%s%s%s", body, invoker, key)
	t := md5.New()
	io.WriteString(t, buf)
	return fmt.Sprintf("%x", t.Sum(nil))
}
