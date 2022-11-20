package handlers

import (
	"log"
	"net/http"
)

type Index struct {
	l *log.Logger
}

func NewIndex(l *log.Logger) *Index {
	return &Index{l}
}

func (i *Index) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}
