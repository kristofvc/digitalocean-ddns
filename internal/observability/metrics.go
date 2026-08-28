package observability

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Metrics struct {
	mu                        sync.RWMutex
	Checks, Updates, Failures uint64
	LastCheck, LastUpdate     time.Time
	LastSucceeded             bool
	Ready                     bool
}

func (m *Metrics) Check(ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Checks++
	m.LastSucceeded = ok
	if ok {
		m.LastCheck = time.Now()
		m.Ready = true
	}
}
func (m *Metrics) Update(ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ok {
		m.Updates++
		m.LastUpdate = time.Now()
	} else {
		m.Failures++
	}
}
func (m *Metrics) IsReady() bool { m.mu.RLock(); defer m.mu.RUnlock(); return m.Ready }
func (m *Metrics) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	success := 0
	if m.LastSucceeded {
		success = 1
	}
	fmt.Fprintf(w, "# TYPE ddns_ip_checks_total counter\nddns_ip_checks_total %d\n# TYPE ddns_dns_updates_total counter\nddns_dns_updates_total %d\n# TYPE ddns_dns_update_failures_total counter\nddns_dns_update_failures_total %d\n# TYPE ddns_last_successful_check_timestamp_seconds gauge\nddns_last_successful_check_timestamp_seconds %d\n# TYPE ddns_last_successful_update_timestamp_seconds gauge\nddns_last_successful_update_timestamp_seconds %d\n# TYPE ddns_last_check_success gauge\nddns_last_check_success %d\n", m.Checks, m.Updates, m.Failures, m.LastCheck.Unix(), m.LastUpdate.Unix(), success)
}
