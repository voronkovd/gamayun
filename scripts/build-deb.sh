#!/usr/bin/env bash
# Build a .deb from an already-built linux binary.
# Usage: scripts/build-deb.sh <version> <amd64|arm64> <path-to-binary>
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <version> <amd64|arm64> <binary>" >&2
  exit 1
fi

VERSION="${1#v}"
ARCH="$2"
BIN="$3"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

case "$ARCH" in
  amd64|arm64) ;;
  *)
    echo "unsupported arch: $ARCH" >&2
    exit 1
    ;;
esac

if [[ ! -f "$BIN" ]]; then
  echo "binary not found: $BIN" >&2
  exit 1
fi

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

mkdir -p \
  "$STAGE/DEBIAN" \
  "$STAGE/usr/bin" \
  "$STAGE/lib/systemd/system" \
  "$STAGE/usr/share/doc/gamayun" \
  "$STAGE/etc/gamayun" \
  "$STAGE/var/lib/gamayun"

install -m 0755 "$BIN" "$STAGE/usr/bin/gamayun"
sed 's|/usr/local/bin/gamayun|/usr/bin/gamayun|' \
  "$ROOT/deploy/gamayun.service" \
  > "$STAGE/lib/systemd/system/gamayun.service"
install -m 0644 "$ROOT/configs/config.example.yaml" "$STAGE/usr/share/doc/gamayun/config.example.yaml"
install -m 0644 "$ROOT/LICENSE" "$STAGE/usr/share/doc/gamayun/copyright"

cat >"$STAGE/DEBIAN/control" <<EOF
Package: gamayun
Version: $VERSION
Section: admin
Priority: optional
Architecture: $ARCH
Maintainer: Dmitry Voronkov <dmitryvoronkov@users.noreply.github.com>
Depends: systemd
Description: Gamayun VPS health agent with Telegram alerts
 Long-running daemon that checks nginx, TLS, disk, memory, load and Docker,
 sends instant Telegram alerts and a daily digest.
EOF

cat >"$STAGE/DEBIAN/postinst" <<'EOF'
#!/bin/sh
set -e
mkdir -p /etc/gamayun /var/lib/gamayun
if [ ! -f /etc/gamayun/config.yaml ]; then
  cp /usr/share/doc/gamayun/config.example.yaml /etc/gamayun/config.yaml
  chmod 600 /etc/gamayun/config.yaml
fi
if [ -d /run/systemd/system ]; then
  systemctl daemon-reload >/dev/null || true
  systemctl enable gamayun.service >/dev/null || true
  systemctl restart gamayun.service >/dev/null || true
fi
EOF

cat >"$STAGE/DEBIAN/prerm" <<'EOF'
#!/bin/sh
set -e
if [ "$1" = "remove" ] && [ -d /run/systemd/system ]; then
  systemctl disable --now gamayun.service >/dev/null || true
fi
EOF

cat >"$STAGE/DEBIAN/postrm" <<'EOF'
#!/bin/sh
set -e
if [ "$1" = "purge" ]; then
  rm -f /etc/gamayun/config.yaml
  rmdir /etc/gamayun 2>/dev/null || true
fi
EOF

chmod 0755 "$STAGE/DEBIAN/postinst" "$STAGE/DEBIAN/prerm" "$STAGE/DEBIAN/postrm"

mkdir -p "$ROOT/dist"
OUT="$ROOT/dist/gamayun_${VERSION}_${ARCH}.deb"
dpkg-deb --root-owner-group --build "$STAGE" "$OUT"
echo "built $OUT"
