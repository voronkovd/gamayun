# Проверки

Каждый тик `checks.Runner` собирает `[]Result`. Ниже — ключ, источник, порог, когда результат `Skip`, и текст алерта. Пороги задаются в YAML, см. [configs/config.example.yaml](../configs/config.example.yaml).

Общее: `Skip` не алертит и не открывает инцидент. Ошибка чтения источника при включённой проверке — это FAIL, не Skip.

## nginx.active

| | |
|---|---|
| Источник | `systemctl is-active nginx` |
| FAIL | состояние не `active` |
| Skip | `checks.nginx: off`; либо `auto` и unit нет (`LoadState=not-found`) |
| Метрики | `state` |
| Текст | `nginx: NOT active (state: failed)` |

`checks.nginx: on` всегда проверяет: нет unit — тоже FAIL.

Как понять, что unit есть: `systemctl show nginx.service -p LoadState`. Значение `not-found` → для `auto` обе nginx-проверки пропускаются.

## nginx.port.80 / nginx.port.443

| | |
|---|---|
| Источник | `/proc/net/tcp` и `/proc/net/tcp6`, состояние `0A` (LISTEN), порт в hex |
| FAIL | нет LISTEN на `:80` или `:443` (любой адрес, IPv4 или IPv6) |
| Skip | те же условия, что у `nginx.active` |
| Метрики | `listening=yes\|no` |
| Текст | `nginx: not listening on :80` |

`ss` не вызываем. Слушающий сокет на конкретном IP считается достаточным.

## cert.\<name\>

| | |
|---|---|
| Источник | `/etc/letsencrypt/live/*/cert.pem`, разбор `crypto/x509` |
| Порог | дней до `NotAfter` &lt; `checks.cert_days_min` (дефолт 14) |
| Skip | каталога нет или в нём нет `cert.pem` — одна запись `certs` со `Skip` |
| FAIL | файл не читается / не парсится; либо дней меньше порога |
| Метрики | `days`, `not_after` |
| Текст | `cert example.com: expires in 3d (2026-09-02 12:00:00 +0000 UTC)` или `cert example.com: cannot read expiry` |

`name` — имя каталога в `live/`.

## disk.root

| | |
|---|---|
| Источник | `syscall.Statfs("/")` |
| Формула % | как GNU `df`: `used = blocks - bfree`, `pct = used * 100 / (used + bavail)` |
| FAIL | `pct >= checks.disk_pct_max` |
| Skip | нет |
| Метрики | `pct`, `max` |
| Текст | `disk /: 91% used (>=85%)` |

## mem.available

| | |
|---|---|
| Источник | `/proc/meminfo`, поле `MemAvailable` (килобайты → МБ, деление на 1024) |
| FAIL | доступно меньше `checks.mem_avail_min_mb` |
| Skip | нет; нет `/proc/meminfo` — FAIL |
| Метрики | `mb`, `min` |
| Текст | `RAM: only 80MB available (<150MB)` |

## load.15

| | |
|---|---|
| Источник | `/proc/loadavg`, третье поле |
| FAIL | значение строго больше `checks.load15_max` |
| Skip | нет; нет файла — FAIL |
| Метрики | `load15`, `max` |
| Текст | `load15: 3.21 (>2.0)` |

## docker.unhealthy

| | |
|---|---|
| Источник | `docker ps --filter health=unhealthy --format '{{.Names}}'` |
| FAIL | список непустой |
| Skip | бинарника `docker` нет в `PATH` |
| Метрики | `names` (через запятую) или пусто |
| Текст | `docker unhealthy: foo,bar` |

## docker.state

| | |
|---|---|
| Источник | `docker ps -a --format '{{.Names}} {{.State}}'` |
| FAIL | state равен `restarting`, `exited` или `dead` |
| Skip | нет `docker` |
| Метрики | `names` вида `foo(exited),bar(dead)` |
| Текст | `docker bad state: foo(exited)` |

Остановленный контейнер, которого нет в `checks.containers`, тоже попадёт сюда.

## docker.required.\<name\>

| | |
|---|---|
| Источник | `docker ps --format '{{.Names}}'`, список `checks.containers` |
| FAIL | контейнера нет среди running |
| Skip | нет `docker` или `checks.containers` пуст (тогда результатов нет) |
| Метрики | `running=yes\|no` |
| Текст | `docker: required container 'marzban-marzban-1' not running` |

Таймаут docker-команд — 15 с. Если CLI вернул ошибку при установленном docker — FAIL с текстом ошибки, не Skip.
