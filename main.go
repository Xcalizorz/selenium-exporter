package main

import (
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/xcalizorz/selenium-exporter/handlers"
	"github.com/xcalizorz/selenium-exporter/metrics"
)

func main() {
	addr := "127.0.0.1:8080"
	l := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}

	sm := http.NewServeMux()

	reg := prometheus.NewRegistry()
	gridExporter := metrics.NewGridExporter(l, reg)

	sm.Handle("/", handlers.NewIndex(l))
	sm.Handle("/status", handlers.NewStatus(l))
	sm.Handle("/metrics", gridExporter.Serve(promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg})))

	s := &http.Server{
		Addr:    addr,
		Handler: sm,
		// ErrorLog:     l,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 10,
		IdleTimeout:  time.Second * 120,
	}

	// start the server
	go func() {
		l.Infoln("Starting server @", addr)

		err := s.ListenAndServe()
		if err != nil {
			l.Infoln("Server terminated:", err)
			os.Exit(1)
		}
	}()

	// trap sigterm or interupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)

	// Block until a signal is received.
	sig := <-c
	log.Infoln("Got signal:", sig)

	// gracefully shutdown the server, waiting max 30 seconds for current operations to complete
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	s.Shutdown(ctx)
}
