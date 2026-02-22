package usersclient

import (
	"strconv"
	"time"
)

func newRequestID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
