package handlers

import (
	"io"
	"log"
	"net/http"
)

type Hello struct {
	l *log.Logger
}

func NewStatus(l *log.Logger) *Hello {
	return &Hello{l}
}

func (h *Hello) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
}
