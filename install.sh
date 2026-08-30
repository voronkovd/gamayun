#!/usr/bin/env bash
set -euo pipefail

if [[ ${EUID:-0} -ne 0 ]]; then
  echo "run as root" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY="$ROOT/deploy"
EXAMPLE="$ROOT/configs/config.example.yaml"
CONF_DIR=/etc/gamayun
CONF_FILE="$CONF_DIR/config.yaml"

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) goarch=amd64 ;;
  aarch64|arm64) goarch=arm64 ;;
  *)
    echo "unsupported arch: $arch" >&2
    exit 1
    ;;
esac

bin=""
for cand in \
  "$ROOT/dist/gamayun-linux-$goarch" \
  "$ROOT/dist/gamayun"; do
  if [[ -x "$cand" ]]; then
    bin="$cand"
    break
  fi
done

if [[ -z "$bin" ]]; then
  if command -v go >/dev/null 2>&1; then
    mkdir -p "$ROOT/dist"
    (
      cd "$ROOT"
      GOOS=linux GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -o "$ROOT/dist/gamayun-linux-$goarch" ./cmd/gamayun
    )
    bin="$ROOT/dist/gamayun-linux-$goarch"
  else
    echo "no binary in dist/ and no go compiler" >&2
    exit 1
  fi
fi

install -m 0755 "$bin" /usr/local/bin/gamayun
install -d -m 0755 "$CONF_DIR" /var/lib/gamayun

if [[ ! -f "$CONF_FILE" ]]; then
  install -m 0600 "$EXAMPLE" "$CONF_FILE"
  echo "Created $CONF_FILE — edit telegram.bot_token and telegram.chat_id"
fi
chmod 0600 "$CONF_FILE"

install -m 0644 "$DEPLOY/gamayun.service" /etc/systemd/system/gamayun.service
systemctl daemon-reload
systemctl enable --now gamayun

echo "Installed. Config: $CONF_FILE"
echo "Test: /usr/local/bin/gamayun --test"
if ! /usr/local/bin/gamayun --test; then
  echo "telegram test failed — edit $CONF_FILE and retry" >&2
fi
