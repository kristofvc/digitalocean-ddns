package publicip

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

type Client struct {
	HTTP      *http.Client
	Providers []string
}

func (c Client) Get(ctx context.Context) (net.IP, error) {
	var errors []string
	for _, url := range c.Providers {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 128))
		resp.Body.Close()
		if readErr != nil || resp.StatusCode/100 != 2 {
			errors = append(errors, fmt.Sprintf("%s returned %s", url, resp.Status))
			continue
		}
		ip := net.ParseIP(strings.TrimSpace(string(body)))
		if ip == nil || ip.To4() == nil {
			errors = append(errors, fmt.Sprintf("%s returned invalid IPv4", url))
			continue
		}
		return ip.To4(), nil
	}
	return nil, fmt.Errorf("all public IP providers failed: %s", strings.Join(errors, "; "))
}
