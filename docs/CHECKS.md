# Checks

Each tick `checks.Runner` collects `[]Result`. Below: key, source, threshold, when the result is `Skip`, and the alert text. Thresholds are set in YAML, see [configs/config.example.yaml](../configs/config.example.yaml).

In general: `Skip` does not alert and does not open an incident. A read error on an enabled check is FAIL, not Skip.

## nginx.active

| | |
|---|---|
| Source | `systemctl is-active nginx` |
| FAIL | state is not `active` |
| Skip | `checks.nginx: off`; or `auto` and the unit is missing (`LoadState=not-found`) |
| Metrics | `state` |
| Text | `nginx: NOT active (state: failed)` |

`checks.nginx: on` always checks: no unit is also FAIL.

How to tell the unit exists: `systemctl show nginx.service -p LoadState`. Value `not-found` → both nginx checks are skipped in `auto`.

## nginx.port.80 / nginx.port.443

| | |
|---|---|
| Source | `/proc/net/tcp` and `/proc/net/tcp6`, state `0A` (LISTEN), port in hex |
| FAIL | no LISTEN on `:80` or `:443` (any address, IPv4 or IPv6) |
| Skip | same conditions as `nginx.active` |
| Metrics | `listening=yes\|no` |
| Text | `nginx: not listening on :80` |

`ss` is not used. A listening socket on a specific IP is enough.

## cert.\<name\>

| | |
|---|---|
| Source | `/etc/letsencrypt/live/*/cert.pem`, parsed with `crypto/x509` |
| Threshold | days until `NotAfter` &lt; `checks.cert_days_min` (default 14) |
| Skip | directory missing or no `cert.pem` in it — one `certs` result with `Skip` |
| FAIL | file cannot be read / parsed; or days below the threshold |
| Metrics | `days`, `not_after` |
| Text | `cert example.com: expires in 3d (2026-09-02 12:00:00 +0000 UTC)` or `cert example.com: cannot read expiry` |

`name` is the directory name under `live/`.

## disk.root

| | |
|---|---|
| Source | `syscall.Statfs("/")` |
| % formula | same as GNU `df`: `used = blocks - bfree`, `pct = used * 100 / (used + bavail)` |
| FAIL | `pct >= checks.disk_pct_max` |
| Skip | never |
| Metrics | `pct`, `max` |
| Text | `disk /: 91% used (>=85%)` |

## mem.available

| | |
|---|---|
| Source | `/proc/meminfo`, field `MemAvailable` (kilobytes → MB, divide by 1024) |
| FAIL | available is below `checks.mem_avail_min_mb` |
| Skip | never; no `/proc/meminfo` — FAIL |
| Metrics | `mb`, `min` |
| Text | `RAM: only 80MB available (<150MB)` |

## load.15

| | |
|---|---|
| Source | `/proc/loadavg`, third field |
| FAIL | value is strictly greater than `checks.load15_max` |
| Skip | never; no file — FAIL |
| Metrics | `load15`, `max` |
| Text | `load15: 3.21 (>2.0)` |

## docker.unhealthy

| | |
|---|---|
| Source | `docker ps --filter health=unhealthy --format '{{.Names}}'` |
| FAIL | list is non-empty |
| Skip | `docker` binary is not in `PATH` |
| Metrics | `names` (comma-separated) or empty |
| Text | `docker unhealthy: foo,bar` |

## docker.state

| | |
|---|---|
| Source | `docker ps -a --format '{{.Names}}\|{{.State}}\|{{.Status}}'` |
| FAIL | state is `restarting` or `dead`; or `exited` with a non-zero exit code |
| Skip | no `docker` |
| Metrics | `names` like `foo(exited),bar(dead)` |
| Text | `docker bad state: foo(exited)` |

A container that exits with code 0 by design (e.g. an init/migration/seed container with `restart: "no"`) is not flagged. A stopped container that is not in `checks.containers` still shows up here if its exit code is non-zero (or the status cannot be parsed).

## docker.required.\<name\>

| | |
|---|---|
| Source | `docker ps --format '{{.Names}}'`, list `checks.containers` |
| FAIL | container is not among the running ones |
| Skip | no `docker`, or `checks.containers` is empty (then there are no results) |
| Metrics | `running=yes\|no` |
| Text | `docker: required container 'marzban-marzban-1' not running` |

Docker command timeout is 15s. If the CLI returns an error while docker is installed — FAIL with the error text, not Skip.
