package updater

import (
	"context"
	"github.com/example/digitalocean-ddns/internal/config"
	"github.com/example/digitalocean-ddns/internal/digitalocean"
	"github.com/example/digitalocean-ddns/internal/observability"
	"io"
	"log/slog"
	"net"
	"testing"
)

type ips struct{}

func (ips) Get(context.Context) (net.IP, error) { return net.ParseIP("203.0.113.9"), nil }

type dns struct {
	data    string
	updates int
}

func (d *dns) FindARecord(context.Context, string, string) (digitalocean.Record, error) {
	return digitalocean.Record{ID: 1, Name: "home", Data: d.data, TTL: 300}, nil
}
func (d *dns) UpdateRecord(_ context.Context, _ string, _ digitalocean.Record, ip string) error {
	d.updates++
	d.data = ip
	return nil
}
func TestUpdatesOnlyWhenChanged(t *testing.T) {
	d := &dns{data: "192.0.2.1"}
	m := &observability.Metrics{}
	u := Updater{IP: ips{}, DNS: d, Records: []config.Record{{Zone: "example.com", Name: "home"}}, Metrics: m, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := u.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := u.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if d.updates != 1 {
		t.Fatalf("updates=%d", d.updates)
	}
}
