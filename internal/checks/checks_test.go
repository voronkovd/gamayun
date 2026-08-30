package checks

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/voronkovd/gamayun/internal/config"
)

func TestMemoryAndLoadFromProc(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "meminfo"), []byte("MemAvailable: 102400 kB\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "loadavg"), []byte("0.1 0.2 0.3 1/2 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.ProcFS = dir
	cfg.MemAvailMinMB = 50
	cfg.Load15Max = 1.0

	mem := Memory{Cfg: cfg}.Run(context.Background())
	if len(mem) != 1 || !mem[0].OK || mem[0].Metrics["mb"] != "100" {
		t.Fatalf("mem: %+v", mem)
	}
	load := Load{Cfg: cfg}.Run(context.Background())
	if len(load) != 1 || !load[0].OK {
		t.Fatalf("load: %+v", load)
	}

	cfg.MemAvailMinMB = 200
	mem = Memory{Cfg: cfg}.Run(context.Background())
	if mem[0].OK {
		t.Fatal("expected mem fail")
	}
}

func TestPortsFromProc(t *testing.T) {
	dir := t.TempDir()
	netDir := filepath.Join(dir, "net")
	if err := os.MkdirAll(netDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tcp := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000:0050 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 1
`
	if err := os.WriteFile(filepath.Join(netDir, "tcp"), []byte(tcp), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults()
	cfg.ProcFS = dir
	cfg.NginxEnabled = "1"
	exec := Exec{
		LookPath: func(string) (string, error) { return "/bin/systemctl", nil },
		Run: func(ctx context.Context, name string, args ...string) (string, error) {
			if len(args) >= 1 && args[0] == "show" {
				return "LoadState=loaded", nil
			}
			return "active", nil
		},
	}
	got := Ports{Cfg: cfg, Exec: exec}.Run(context.Background())
	if len(got) != 2 {
		t.Fatalf("got %d", len(got))
	}
	if !got[0].OK || got[0].Key != "nginx.port.80" {
		t.Fatalf("80: %+v", got[0])
	}
	if got[1].OK || got[1].Key != "nginx.port.443" {
		t.Fatalf("443: %+v", got[1])
	}
}

func TestCertsExpiry(t *testing.T) {
	dir := t.TempDir()
	live := filepath.Join(dir, "example.com")
	if err := os.MkdirAll(live, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	writeTestCert(t, filepath.Join(live, "cert.pem"), now.Add(3*24*time.Hour))

	cfg := config.Defaults()
	cfg.LetsEncrypt = dir
	cfg.CertDaysMin = 14
	got := Certs{Cfg: cfg, Now: func() time.Time { return now }}.Run(context.Background())
	if len(got) != 1 || got[0].OK || got[0].Key != "cert.example.com" {
		t.Fatalf("%+v", got)
	}
}

func TestDockerRequired(t *testing.T) {
	cfg := config.Defaults()
	cfg.Containers = []string{"must"}
	exec := Exec{
		LookPath: func(string) (string, error) { return "/usr/bin/docker", nil },
		Run: func(ctx context.Context, name string, args ...string) (string, error) {
			joined := args[0]
			if len(args) > 1 {
				joined = args[0] + " " + args[1]
			}
			switch {
			case args[0] == "ps" && len(args) > 1 && args[1] == "--filter":
				return "", nil
			case args[0] == "ps" && len(args) > 1 && args[1] == "-a":
				return "other exited", nil
			case args[0] == "ps":
				return "other", nil
			}
			return joined, nil
		},
	}
	got := Docker{Cfg: cfg, Exec: exec}.Run(context.Background())
	var required, state Result
	for _, r := range got {
		if r.Key == "docker.required.must" {
			required = r
		}
		if r.Key == "docker.state" {
			state = r
		}
	}
	if required.OK {
		t.Fatalf("required should fail: %+v", got)
	}
	if state.OK {
		t.Fatalf("state should fail: %+v", got)
	}
}

func writeTestCert(t *testing.T, path string, notAfter time.Time) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "example.com"},
		NotBefore:    notAfter.Add(-30 * 24 * time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		t.Fatal(err)
	}
}
