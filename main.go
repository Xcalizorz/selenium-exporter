package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/context"

	"github.com/xcalizorz/selenium-exporter/exporter"
	"github.com/xcalizorz/selenium-exporter/handlers"
)

type exporterFlags struct {
	publishAddr string
	seleniumUri string
	version     int
}

func setFlags(l *log.Logger) exporterFlags {
	var result exporterFlags

	flag.StringVar(&result.publishAddr, "listen-address", ":8081", "Address on which to expose metrics.")
	flag.StringVar(&result.seleniumUri, "scrape-uri", "", "URI on which to scrape Selenium Grid - can be set via env. variable 'SE_NODE_GRID_URL'.")
	flag.IntVar(&result.version, "version", 4, "The major version of your Selenium deployment [3, 4]")
	flag.Parse()

	if result.seleniumUri != "" {
		os.Setenv("SE_NODE_GRID_URL", result.seleniumUri)
	}
	if os.Getenv("SE_NODE_GRID_URL") == "" {
		l.Fatal("Provide the URL to your Selenium deployment either via flag or via env. variable 'SE_NODE_GRID_URL'")
	}
	checkUrl(l, os.Getenv("SE_NODE_GRID_URL"))
	os.Setenv("SE_NODE_GRID_VERSION", strconv.Itoa(result.version))

	return result
}

func checkUrl(l *log.Logger, uri string) {
	_, err := url.ParseRequestURI(uri)
	if err != nil {
		l.Fatal(err)
	}
}

func main() {
	l := log.New(os.Stdout, "selenium-exporter", log.LstdFlags)
	f := setFlags(l)

	sm := http.NewServeMux()

	gridExporter := exporter.NewGridExporter(l)
	m := handlers.NewMetrics(l)

	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	sm.Handle("/", handlers.NewIndex(l))
	sm.Handle("/status", handlers.NewStatus(l))
	sm.Handle("/metrics", m.Serve(promhttp.Handler(), gridExporter))

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
		l.Println("Starting server @", f.publishAddr)

		err := s.ListenAndServe()
		if err != nil {
			l.Println("Server terminated:", err)
			os.Exit(1)
		}
	}()

	// trap sigterm or interupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)

	// Block until a signal is received.
	sig := <-c
	l.Println("Got signal:", sig)

	// gracefully shutdown the server, waiting max 30 seconds for current operations to complete
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	s.Shutdown(ctx)
}
