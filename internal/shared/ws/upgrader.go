package ws

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

func BuildUpgrader(isProd bool, allowedOrigins string) websocket.Upgrader {
	allowed := parseOrigins(allowedOrigins)
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if !isProd {
				return true
			}
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			_, ok := allowed[origin]
			return ok
		},
	}
}

func parseOrigins(raw string) map[string]struct{} {
	allowed := make(map[string]struct{})
	for o := range strings.SplitSeq(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[o] = struct{}{}
		}
	}
	return allowed
}
