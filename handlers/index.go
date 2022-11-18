package handlers

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

type Index struct {
	l *logrus.Logger
}

func NewIndex(l *logrus.Logger) *Index {
	return &Index{l}
}

func (i *Index) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}
