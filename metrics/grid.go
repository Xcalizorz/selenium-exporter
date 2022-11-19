package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

type GridExporter struct {
	l                *logrus.Logger
	maxSession       prometheus.Gauge
	sessionCount     prometheus.Gauge
	totalSlots       prometheus.Gauge
	nodeCount        prometheus.Gauge
	sessionQueueSize prometheus.Gauge
	version          *prometheus.GaugeVec
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

func NewGridExporter(l *logrus.Logger) *GridExporter {
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

func (e *GridExporter) Serve(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uri := fmt.Sprintf("%s/graphql", os.Getenv("SE_NODE_GRID_URL"))
		err := e.fetch(uri)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func (e *GridExporter) fetch(uri string) error {
	e.l.Infoln("Fetching data from", uri)

	jsonData := map[string]string{
		"query": `
            { 
                grid {
					uri,
					totalSlots,
					nodeCount,
					maxSession,
					sessionCount,
					version,
					sessionQueueSize
                }
				nodesInfo {
					nodes{
						id,
						uri,
						status,
						maxSession,
						slotCount,
						sessions {
								id,
								capabilities,
								startTime,
								uri,
								nodeId,
								nodeUri,
								sessionDurationMillis
								slot {
									id,
									stereotype,
									lastStarted
								}
						},
						sessionCount,
						stereotypes,
						version,
						osInfo {
							arch,
							name,
							version
						}
					}
				}
            }
        `,
	}
	jsonValue, _ := json.Marshal(jsonData)

	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewBuffer(jsonValue))
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.l.Errorf("Unable to get data from Selenium @ %s: %s", uri, err)
		return err
	}

	decoder := json.NewDecoder(resp.Body)
	jsonBody := Body{}
	err = decoder.Decode(&jsonBody)
	if err != nil {
		e.l.Errorf("Can not unmarshal response body", resp.Body)
		return err
	}

	e.maxSession.Set(float64(jsonBody.Data.Grid.MaxSession))
	e.sessionCount.Set(float64(jsonBody.Data.Grid.SessionCount))
	e.nodeCount.Set(float64(jsonBody.Data.Grid.NodeCount))
	e.sessionQueueSize.Set(float64(jsonBody.Data.Grid.SessionQueueSize))
	e.totalSlots.Set(float64(jsonBody.Data.Grid.TotalSlots))
	e.version.WithLabelValues(jsonBody.Data.Grid.Version).Set(float64(1))

	return nil
}
