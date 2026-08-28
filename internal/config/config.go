package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Record struct{ Zone, Name string }

type Config struct {
	Token         string
	Records       []Record
	Interval      time.Duration
	ListenAddress string
	Providers     []string
	DOAPIBaseURL  string
}

func Load() (Config, error) {
	c := Config{Token: os.Getenv("DO_TOKEN"), ListenAddress: env("HTTP_LISTEN_ADDRESS", ":8080"), DOAPIBaseURL: env("DO_API_BASE_URL", "https://api.digitalocean.com/v2")}
	if c.Token == "" {
		return c, fmt.Errorf("DO_TOKEN is required")
	}
	var err error
	c.Interval, err = time.ParseDuration(env("POLL_INTERVAL", "60s"))
	if err != nil || c.Interval <= 0 {
		return c, fmt.Errorf("POLL_INTERVAL must be a positive duration")
	}
	for _, item := range strings.Split(os.Getenv("DNS_RECORDS"), ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return c, fmt.Errorf("invalid DNS_RECORDS item %q (want zone/name)", item)
		}
		c.Records = append(c.Records, Record{Zone: parts[0], Name: parts[1]})
	}
	if len(c.Records) == 0 {
		return c, fmt.Errorf("DNS_RECORDS is required")
	}
	for _, p := range strings.Split(env("PUBLIC_IP_PROVIDERS", "https://api.ipify.org,https://checkip.amazonaws.com"), ",") {
		if p = strings.TrimSpace(p); p != "" {
			c.Providers = append(c.Providers, p)
		}
	}
	if len(c.Providers) == 0 {
		return c, fmt.Errorf("at least one PUBLIC_IP_PROVIDERS entry is required")
	}
	return c, nil
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
func LookupTimeout() time.Duration {
	v, _ := strconv.Atoi(env("HTTP_TIMEOUT_SECONDS", "10"))
	if v <= 0 {
		v = 10
	}
	return time.Duration(v) * time.Second
}
