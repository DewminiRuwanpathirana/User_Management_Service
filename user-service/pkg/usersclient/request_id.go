package usersclient

import (
	"strconv"
	"time"
)

func newRequestID() string {
	// convert an int64 number to string and format it as a base-10 string.
	return strconv.FormatInt(time.Now().UnixNano(), 10)
}
