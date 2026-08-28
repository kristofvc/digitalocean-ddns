# DigitalOcean DDNS

A small, continuously running Go service that discovers the host's public IPv4 address and reconciles one or more existing DigitalOcean A records. It updates only records whose value differs, retries temporary failures on the next cycle, emits JSON logs without secrets, and shuts down cleanly on SIGINT/SIGTERM.

## Architecture

The updater tries public-IP providers in order, then queries each configured record through DigitalOcean API v2 and updates changed values. An independent HTTP server exposes health, readiness, and metrics. There is no database or Kubernetes ownership: deployment belongs in the separate GitOps repository.

## Configuration

| Variable | Required | Default | Meaning |
|---|---:|---|---|
| `DO_TOKEN` | yes | — | DigitalOcean API token; never logged |
| `DNS_RECORDS` | yes | — | Comma-separated `zone/name` pairs; use `@` for apex |
| `POLL_INTERVAL` | no | `60s` | Positive Go duration |
| `HTTP_LISTEN_ADDRESS` | no | `:8080` | Observability listen address |
| `HTTP_TIMEOUT_SECONDS` | no | `10` | Timeout for external HTTP requests |
| `PUBLIC_IP_PROVIDERS` | no | ipify, AWS checkip | Comma-separated endpoint URLs, tried in order |
| `DO_API_BASE_URL` | no | DigitalOcean API v2 | Primarily useful for testing |

The token needs permission to read and update domain records for every configured zone. Prefer a scoped token with only the DNS access the account supports.

## Run

```sh
DO_TOKEN='...' DNS_RECORDS='example.com/home,example.net/@' go run ./cmd/digitalocean-ddns
```

```sh
docker build -t digitalocean-ddns .
docker run --rm -p 8080:8080 -e DO_TOKEN -e DNS_RECORDS='example.com/home' digitalocean-ddns
```

Providers are attempted sequentially; invalid responses, non-2xx responses, and network failures fall through to the next provider. If all fail, the cycle is marked failed and retried later.

## Observability

- `/healthz` is live whenever the HTTP process can respond.
- `/readyz` returns 503 until the first successful public-IP check, then 200.
- `/metrics` exports `ddns_ip_checks_total`, `ddns_dns_updates_total`, `ddns_dns_update_failures_total`, successful check/update Unix timestamps, and `ddns_last_check_success`. The IP itself is deliberately absent.

## Build and test

```sh
gofmt -w .
go vet ./...
go test -race ./...
go build ./cmd/digitalocean-ddns
```

## Releases and GHCR

Pull requests run formatting, vet, tests, and builds. Pushes to `main` publish `ghcr.io/<owner>/<repository>:sha-<full-commit-sha>`. Pushing a semantic tag such as `v0.1.0` additionally publishes `:0.1.0`; no mutable `latest` tag is emitted. Images are built for linux/amd64 and linux/arm64 using GitHub's `GITHUB_TOKEN` with only `contents: read` and `packages: write` in the publishing job.

## Kubernetes example

[`examples/kubernetes/deployment.yaml`](examples/kubernetes/deployment.yaml) demonstrates environment configuration, secret injection, probes, and a pinned version. It is intentionally only an example: copy/adapt it in the separate Argo CD-managed homelab repository, where the real Deployment, Secret, ServiceMonitor, and related resources belong. Never commit the token.
