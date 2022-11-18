package metrics

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

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
	Data Data `json:"data"`
}

type Data struct {
	Grid      Grid      `json:"grid"`
	NodesInfo NodesInfo `json:"nodesInfo"`
}

type NodesInfo struct {
	Nodes []Node `json:"nodes"`
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
	OsInfo       OsInfo    `json:"osInfo"`
	Sessions     []Session `json:"sessions"`
}

type OsInfo struct {
	Arch    string `json:"arch"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Session struct {
	Id                    string `json:"id"`
	Capabilities          string `json:"capabilities"`
	StartTime             string `json:"startTime"`
	Uri                   string `json:"uri"`
	NodeId                string `json:"nodeId"`
	NodeUri               string `json:"nodeUri"`
	SessionDurationMillis string `json:"sessionDurationMillis"`
	Slot                  Slot   `json:"slot"`
}

type Slot struct {
	Id          string `json:"id"`
	Stereotype  string `json:"sterotype"`
	LastStarted string `json:"lastStarted"`
}

type Grid struct {
	Uri              string `json:"-"`
	MaxSession       int64  `json:"maxSession"`
	SessionCount     int64  `json:"sessionCount"`
	TotalSlots       int64  `json:"totalSlots"`
	NodeCount        int64  `json:"nodeCount"`
	Version          string `json:"version"`
	SessionQueueSize int64  `json:"sessionQueueSize"`
}

func NewGridExporter(l *logrus.Logger, reg prometheus.Registerer) *GridExporter {
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
	reg.MustRegister(e.maxSession)
	reg.MustRegister(e.sessionCount)
	reg.MustRegister(e.totalSlots)
	reg.MustRegister(e.nodeCount)
	reg.MustRegister(e.sessionQueueSize)
	reg.MustRegister(e.version)
	return e
}

func (e *GridExporter) Serve(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.fetch("https://mykn-shared-selenium-4.apps.emea.ocp.int.kn/graphql")
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

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	// e.l.Debugf("%s", body)

	if resp.StatusCode >= 400 {
		e.l.Errorf("Unable to get data from Selenium @ %s: %s", uri, body)
	}

	if err != nil {
		e.l.Errorf(`Unable to parse response from Selenium "%s": %s`, body, err)
		return err
	}

	var jsonBody Body
	json.Unmarshal(body, &jsonBody)

	e.maxSession.Set(float64(jsonBody.Data.Grid.MaxSession))
	e.sessionCount.Set(float64(jsonBody.Data.Grid.SessionCount))
	e.nodeCount.Set(float64(jsonBody.Data.Grid.NodeCount))
	e.sessionQueueSize.Set(float64(jsonBody.Data.Grid.SessionQueueSize))
	e.totalSlots.Set(float64(jsonBody.Data.Grid.TotalSlots))
	e.version.WithLabelValues(jsonBody.Data.Grid.Version).Set(float64(1))

	return nil
}
