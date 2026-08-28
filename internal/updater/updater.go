package updater

import (
	"context"
	"fmt"
	"github.com/example/digitalocean-ddns/internal/config"
	"github.com/example/digitalocean-ddns/internal/digitalocean"
	"github.com/example/digitalocean-ddns/internal/observability"
	"log/slog"
	"net"
	"time"
)

type IPSource interface {
	Get(context.Context) (net.IP, error)
}
type DNSClient interface {
	FindARecord(context.Context, string, string) (digitalocean.Record, error)
	UpdateRecord(context.Context, string, digitalocean.Record, string) error
}
type Updater struct {
	IP      IPSource
	DNS     DNSClient
	Records []config.Record
	Metrics *observability.Metrics
	Log     *slog.Logger
}

func (u Updater) RunOnce(ctx context.Context) error {
	ip, err := u.IP.Get(ctx)
	if err != nil {
		u.Metrics.Check(false)
		return fmt.Errorf("detect public IP: %w", err)
	}
	u.Metrics.Check(true)
	u.Log.Info("public IP check succeeded")
	var first error
	for _, wanted := range u.Records {
		r, err := u.DNS.FindARecord(ctx, wanted.Zone, wanted.Name)
		if err != nil {
			u.Metrics.Update(false)
			u.Log.Error("DNS record lookup failed", "zone", wanted.Zone, "name", wanted.Name, "error", err)
			if first == nil {
				first = err
			}
			continue
		}
		if r.Data == ip.String() {
			u.Log.Debug("DNS record is current", "zone", wanted.Zone, "name", wanted.Name)
			continue
		}
		u.Log.Info("public IP change detected", "zone", wanted.Zone, "name", wanted.Name)
		if err := u.DNS.UpdateRecord(ctx, wanted.Zone, r, ip.String()); err != nil {
			u.Metrics.Update(false)
			u.Log.Error("DNS update failed", "zone", wanted.Zone, "name", wanted.Name, "error", err)
			if first == nil {
				first = err
			}
			continue
		}
		u.Metrics.Update(true)
		u.Log.Info("DNS record updated", "zone", wanted.Zone, "name", wanted.Name)
	}
	return first
}
func (u Updater) Run(ctx context.Context, interval time.Duration) {
	u.check(ctx)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			u.check(ctx)
		}
	}
}
func (u Updater) check(ctx context.Context) {
	if err := u.RunOnce(ctx); err != nil {
		u.Log.Error("update cycle failed; will retry", "error", err)
	}
}
