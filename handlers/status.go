package handlers

import (
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

type Hello struct {
	l *logrus.Logger
}

func NewStatus(l *logrus.Logger) *Hello {
	return &Hello{l}
}

func (h *Hello) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
}
