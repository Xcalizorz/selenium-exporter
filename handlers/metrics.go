package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/xcalizorz/selenium-exporter/exporter"
)

type Metrics struct {
	l *log.Logger
}

func NewMetrics(l *log.Logger) *Metrics {
	return &Metrics{l}
}

func (m *Metrics) Serve(h http.Handler, e *exporter.GridExporter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uri := fmt.Sprintf("%s/graphql", os.Getenv("SE_NODE_GRID_URL"))
		b, status := m.fetch(uri)
		e.SetMetrics(b)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func generateGraphqlQuery(version string) (map[string]string, error) {
	if version == "4" {
		return map[string]string{
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
					sessionsInfo {
						sessionQueueRequests,
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
						}
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
		}, nil
	}
	if version == "3" {
		return map[string]string{}, nil
	}

	return nil, errors.New("Version must be set to '3' or '4'")
}

func (m *Metrics) fetch(uri string) (exporter.Body, int) {
	m.l.Println("Fetching data from", uri)

	jsonData, err := generateGraphqlQuery(os.Getenv("SE_NODE_GRID_VERSION"))
	if err != nil {
		m.l.Fatal(err)
		return exporter.Body{}, http.StatusFailedDependency
	}
	jsonValue, _ := json.Marshal(jsonData)

	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewBuffer(jsonValue))
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		m.l.Printf("Unable to get data from Selenium @ %s: %s", uri, err)
		return exporter.Body{}, http.StatusFailedDependency
	}

	decoder := json.NewDecoder(resp.Body)
	b := exporter.Body{}
	err = decoder.Decode(&b)
	if err != nil {
		m.l.Println("Can not unmarshal response body", resp.Body)
		return exporter.Body{}, http.StatusBadRequest
	}

	return b, http.StatusOK
}
