package utils

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
)

func NewRandomToken() (string, error) {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func GetClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func GetUserAgent(r *http.Request) string {
	return r.UserAgent()
}
