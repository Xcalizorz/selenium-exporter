package exporter

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type GridExporter struct {
	l                    *log.Logger
	maxSession           prometheus.Gauge
	sessionCount         prometheus.Gauge
	totalSlots           prometheus.Gauge
	nodeCount            prometheus.Gauge
	sessionQueueSize     prometheus.Gauge
	version              *prometheus.GaugeVec
	sessionQueueRequests prometheus.Gauge
}

type Body struct {
	Data struct {
		Grid struct {
			Uri              string `json:"-"`
			MaxSession       int64  `json:"maxSession"`
			SessionCount     int64  `json:"sessionCount"`
			TotalSlots       int64  `json:"totalSlots"`
			NodeCount        int64  `json:"nodeCount"`
			Version          string `json:"version"`
			SessionQueueSize int64  `json:"sessionQueueSize"`
		} `json:"grid"`

		SessionInfo struct {
			SessionQueueRequests int64     `json:"sessionQueueRequests"`
			Sessions             []Session `json:"sessions"`
		} `json:"sessionInfo"`

		NodesInfo struct {
			Nodes []Node `json:"nodes"`
		} `json:"nodesInfo"`
	} `json:"data"`
}

type Node struct {
	Id           string    `json:"id"`
	Uri          string    `json:"uri"`
	Status       string    `json:"status"`
	MaxSession   int64     `json:"maxSession"`
	SlotCount    int64     `json:"slotCount"`
	SessionCount int64     `json:"sessionCount"`
	Stereotypes  string    `json:"stereotypes"`
	Version      string    `json:"version"`
	Sessions     []Session `json:"sessions"`
	OsInfo       struct {
		Arch    string `json:"arch"`
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"osInfo"`
}

type Session struct {
	Id                    string `json:"id"`
	Capabilities          string `json:"capabilities"`
	StartTime             string `json:"startTime"`
	Uri                   string `json:"uri"`
	NodeId                string `json:"nodeId"`
	NodeUri               string `json:"nodeUri"`
	SessionDurationMillis string `json:"sessionDurationMillis"`
	Slot                  struct {
		Id          string `json:"id"`
		Stereotype  string `json:"sterotype"`
		LastStarted string `json:"lastStarted"`
	} `json:"slot"`
}

func NewGridExporter(l *log.Logger) *GridExporter {
	e := &GridExporter{
		l: l,
		maxSession: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "selenium_grid_max_session",
			Help: "maximum number of sessions",
		}),
		sessionCount: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "selenium_grid_session_count",
			Help: "number of active sessions",
		}),
		totalSlots: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "selenium_grid_total_slots",
			Help: "maximum number of slots",
		}),
		nodeCount: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "selenium_grid_node_count",
			Help: "number of nodes",
		}),
		sessionQueueSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "selenium_grid_session_queue_size",
			Help: "size of the session queue",
		}),
		sessionQueueRequests: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "selenium_grid_session_queue_requests",
			Help: "number of requests in the session queue",
		}),
		version: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "selenium_grid_version",
				Help: "version of the grid",
			},
			[]string{"version"},
		),
	}
	return e
}

func (e *GridExporter) SetMetrics(b Body) {
	e.maxSession.Set(float64(b.Data.Grid.MaxSession))
	e.sessionCount.Set(float64(b.Data.Grid.SessionCount))
	e.nodeCount.Set(float64(b.Data.Grid.NodeCount))
	e.sessionQueueSize.Set(float64(b.Data.Grid.SessionQueueSize))
	e.totalSlots.Set(float64(b.Data.Grid.TotalSlots))
	e.version.WithLabelValues(b.Data.Grid.Version).Set(float64(1))

	e.sessionQueueRequests.Set(float64(b.Data.SessionInfo.SessionQueueRequests))
}
