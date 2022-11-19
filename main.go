package main

import (
	"flag"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/xcalizorz/selenium-exporter/handlers"
	"github.com/xcalizorz/selenium-exporter/metrics"
)

type exporterFlags struct {
	publishAddr string
	seleniumUri string
	version     int
}

func setFlags(l *logrus.Logger) exporterFlags {
	var result exporterFlags

	flag.StringVar(&result.publishAddr, "addr", ":8081", "The publish address of this exporter")
	flag.StringVar(&result.seleniumUri, "uri", "", "The URL to your Selenium deployment")
	flag.IntVar(&result.version, "version", 4, "The major version of your Selenium deployment [3, 4]")
	flag.Parse()

	checkUrl(l, result.seleniumUri)
	if result.seleniumUri != "" {
		os.Setenv("SE_NODE_GRID_URL", result.seleniumUri)
	}
	if os.Getenv("SE_NODE_GRID_URL") == "" {
		l.Fatal("Provide the URL to your Selenium deployment either via '-uri' or via env. variable 'SE_NODE_GRID_URL'")
		os.Exit(1)
	}
	os.Setenv("SE_NODE_GRID_VERSION", strconv.Itoa(result.version))

	return result
}

func checkUrl(l *logrus.Logger, uri string) {
	_, err := url.ParseRequestURI(uri)
	if err != nil {
		l.Fatal(err)
	}
}

func main() {
	l := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}
	f := setFlags(l)

	sm := http.NewServeMux()

	gridExporter := metrics.NewGridExporter(l)

	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	sm.Handle("/", handlers.NewIndex(l))
	sm.Handle("/status", handlers.NewStatus(l))
	sm.Handle("/metrics", gridExporter.Serve(promhttp.Handler()))

	s := &http.Server{
		Addr:    f.publishAddr,
		Handler: sm,
		// ErrorLog:     l,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 10,
		IdleTimeout:  time.Second * 120,
	}

	// start the server
	go func() {
		l.Infoln("Starting server @", f.publishAddr)

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
