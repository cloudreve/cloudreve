package h3

import (
	"net/http"
	"sync"

	"golang.org/x/sync/syncmap"
)

type server struct {
	handler http.Handler

	conns syncmap.Map
}

func NewServer(api http.Handler) *server {
	return &server{
		handler: api,
		conns:   sync.Map{},
	}
}
