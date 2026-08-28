package digitalocean

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	HTTP           *http.Client
	BaseURL, Token string
}
type Record struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
	Data string `json:"data"`
	TTL  int    `json:"ttl"`
}

func (c Client) FindARecord(ctx context.Context, zone, name string) (Record, error) {
	queryName := name
	if name == "@" {
		queryName = zone
	} else if name != zone && !strings.HasSuffix(name, "."+zone) {
		queryName = name + "." + zone
	}
	u := fmt.Sprintf("%s/domains/%s/records?type=A&name=%s", c.BaseURL, url.PathEscape(zone), url.QueryEscape(queryName))
	var out struct {
		Records []Record `json:"domain_records"`
	}
	if err := c.do(ctx, http.MethodGet, u, nil, &out); err != nil {
		return Record{}, err
	}
	if len(out.Records) == 0 {
		return Record{}, fmt.Errorf("A record %s/%s not found", zone, name)
	}
	if len(out.Records) > 1 {
		return Record{}, fmt.Errorf("A record %s/%s is not unique", zone, name)
	}
	return out.Records[0], nil
}

func (c Client) UpdateRecord(ctx context.Context, zone string, r Record, ip string) error {
	u := fmt.Sprintf("%s/domains/%s/records/%d", c.BaseURL, url.PathEscape(zone), r.ID)
	body := map[string]any{"type": "A", "name": r.Name, "data": ip, "ttl": r.TTL}
	return c.do(ctx, http.MethodPut, u, body, nil)
}

func (c Client) do(ctx context.Context, method, u string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("DigitalOcean API returned %s: %s", resp.Status, string(b))
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
