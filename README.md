# Gamayun

A prophetic bird that warns you. Lightweight VPS monitoring agent written in Go: a systemd service that watches the host, sends Telegram alerts the moment something breaks, and a daily digest of metrics plus incidents.

[![CI](https://github.com/voronkovd/gamayun/actions/workflows/ci.yml/badge.svg)](https://github.com/voronkovd/gamayun/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

More detail: [architecture](docs/ARCHITECTURE.md), [checks](docs/CHECKS.md).

## Features

- Checks every `checks.interval` (default 60s): nginx, ports 80/443, Let's Encrypt certs, disk `/`, free RAM, load15, Docker.
- `PROBLEM` after anti-flap (`alerts.fail_streak`, default 2 consecutive failures).
- Reminders with escalation 5m → 30m → 2h while the issue lasts.
- `RECOVERED` when the check is green again.
- Daily digest at `digest.at` (default 08:00 local time). If the daemon missed the slot, the digest is sent on startup.

One static binary. No Docker SDK, no inbound HTTP.

## Install

### From a .deb

Download from [GitHub Releases](https://github.com/voronkovd/gamayun/releases):

```bash
sudo apt install ./gamayun_1.0.0_amd64.deb
```

Config lands in `/etc/gamayun/config.yaml` and the service starts on its own. Later updates: a new `.deb` or your own apt repo — see [docs/PACKAGING.md](docs/PACKAGING.md).

### From a release binary

Put `gamayun-linux-amd64` or `arm64` into `dist/`, then:

```bash
sudo ./install.sh
sudo gamayun --update   # later versions
```

### From source

Go 1.27.0+.

```bash
make build        # linux/amd64 and linux/arm64 → dist/
sudo ./install.sh
```

`install.sh`:

1. Installs `/usr/local/bin/gamayun`.
2. Creates `/etc/gamayun/config.yaml` from [configs/config.example.yaml](configs/config.example.yaml) if it does not exist.
3. Enables and starts `gamayun.service`.

Then edit the token and chat id:

```bash
sudo ${EDITOR:-nano} /etc/gamayun/config.yaml
sudo chmod 600 /etc/gamayun/config.yaml
sudo systemctl restart gamayun
```

Verify:

```bash
gamayun --test
gamayun --once
journalctl -u gamayun -f
```

## Commands

| Command | What it does |
|---|---|
| `gamayun` | Daemon (systemd mode) |
| `gamayun --once` | One check run to stdout; exit 1 if anything FAILs. No alerts, no state |
| `gamayun --test` | Test Telegram message |
| `gamayun --digest` | Force the daily digest now |
| `gamayun --version` | Version and GitHub repo baked in at build time |
| `gamayun --update` | Fetch the latest release, replace the binary, restart the service |
| `gamayun --config PATH` | Other YAML (default `/etc/gamayun/config.yaml`) |

## Configuration

Live config: **`/etc/gamayun/config.yaml`** (`chmod 600`).  
Commented example: [`configs/config.example.yaml`](configs/config.example.yaml).  
FSM state and incidents: `/var/lib/gamayun/state.json` (do not edit by hand).

```yaml
server_name: my-vps

telegram:
  bot_token: "123456:ABC-DEF..."
  chat_id: "123456789"          # quotes optional

checks:
  interval: 60s
  nginx: auto                   # auto | on | off
  disk_pct_max: 85
  cert_days_min: 14
  mem_avail_min_mb: 150
  load15_max: 2.0
  containers: []

alerts:
  fail_streak: 2
  recover_streak: 1
  escalation: [5m, 30m, 2h]

digest:
  at: "08:00"

paths:
  state: /var/lib/gamayun/state.json
```

Secrets can stay out of the file: `TG_BOT_TOKEN` and `TG_CHAT_ID` override YAML.

## Build and test

```bash
make build        # linux amd64 + arm64
make build-local  # current OS
make test
make deb          # .deb (needs dpkg-deb, Linux)
```

A `v*` tag builds binaries, `.deb` files, and SHA256SUMS.

Updates and apt: [docs/PACKAGING.md](docs/PACKAGING.md).

## License

[MIT](LICENSE)
