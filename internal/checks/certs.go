package checks

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/voronkovd/gamayun/internal/config"
)

type Certs struct {
	Cfg config.Config
	Now func() time.Time
}

func (c Certs) Run(ctx context.Context) []Result {
	_ = ctx
	root := c.Cfg.LetsEncrypt
	if root == "" {
		root = "/etc/letsencrypt/live"
	}
	now := time.Now()
	if c.Now != nil {
		now = c.Now()
	}

	matches, err := filepath.Glob(filepath.Join(root, "*", "cert.pem"))
	if err != nil || len(matches) == 0 {
		return []Result{skipped("certs", "no certificates")}
	}

	var out []Result
	for _, path := range matches {
		name := filepath.Base(filepath.Dir(path))
		key := "cert." + name
		days, notAfter, readErr := certDays(path, now)
		if readErr != nil {
			out = append(out, fail(key, "cert "+name+": cannot read expiry", map[string]string{"days": ""}))
			continue
		}
		metrics := map[string]string{
			"days":      strconv.Itoa(days),
			"not_after": notAfter.Format(time.RFC3339),
		}
		if days < c.Cfg.CertDaysMin {
			out = append(out, fail(key, fmt.Sprintf("cert %s: expires in %dd (%s)", name, days, notAfter.Format("2006-01-02 15:04:05 -0700 MST")), metrics))
			continue
		}
		out = append(out, ok(key, fmt.Sprintf("cert %s: %dd", name, days), metrics))
	}
	return out
}

func certDays(path string, now time.Time) (int, time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, time.Time{}, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return 0, time.Time{}, fmt.Errorf("no pem")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return 0, time.Time{}, err
	}
	return daysUntil(cert.NotAfter, now), cert.NotAfter, nil
}

func daysUntil(notAfter, now time.Time) int {
	return int(notAfter.Sub(now) / (24 * time.Hour))
}
