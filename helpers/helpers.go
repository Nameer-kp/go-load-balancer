package helpers

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}
func GetAttemptsFromContext(r *http.Request) int {
	if attempt, ok := r.Context().Value(Attempts).(int); ok {
		return attempt
	}
	return 0
}
func IsBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		log.Println("Site unreachable, error: ", err)
		return false
	}
	defer conn.Close()
	return true
}
