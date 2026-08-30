package checks

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/voronkovd/gamayun/internal/config"
)

const tcpListen = 0x0A

type Ports struct {
	Cfg  config.Config
	Exec Exec
}

func (p Ports) Run(ctx context.Context) []Result {
	decision := nginxDecision(ctx, p.Cfg, p.Exec)
	if decision.Skip {
		return []Result{
			skipped("nginx.port.80", decision.Reason),
			skipped("nginx.port.443", decision.Reason),
		}
	}

	proc := p.Cfg.ProcFS
	if proc == "" {
		proc = "/proc"
	}
	listening, err := listeningPorts(proc)
	if err != nil {
		msg := "nginx: cannot read listening ports"
		return []Result{
			fail("nginx.port.80", msg, map[string]string{"listening": "error"}),
			fail("nginx.port.443", msg, map[string]string{"listening": "error"}),
		}
	}

	var out []Result
	for _, port := range []int{80, 443} {
		key := "nginx.port." + strconv.Itoa(port)
		if listening[port] {
			out = append(out, ok(key, "nginx: listening on :"+strconv.Itoa(port), map[string]string{"listening": "yes"}))
			continue
		}
		out = append(out, fail(key, "nginx: not listening on :"+strconv.Itoa(port), map[string]string{"listening": "no"}))
	}
	return out
}

func listeningPorts(proc string) (map[int]bool, error) {
	out := map[int]bool{}
	for _, name := range []string{"net/tcp", "net/tcp6"} {
		f, err := os.Open(filepath.Join(proc, name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if err := parseTCPTable(f, out); err != nil {
			f.Close()
			return nil, err
		}
		f.Close()
	}
	if len(out) == 0 {
		if _, err := os.Stat(filepath.Join(proc, "net/tcp")); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func parseTCPTable(r io.Reader, into map[int]bool) error {
	sc := bufio.NewScanner(r)
	first := true
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if first {
			first = false
			if strings.Contains(line, "local_address") {
				continue
			}
		}
		port, listen, ok := parseTCPLine(line)
		if ok && listen {
			into[port] = true
		}
	}
	return sc.Err()
}

func parseTCPLine(line string) (port int, listen bool, ok bool) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return 0, false, false
	}
	local := fields[1]
	colon := strings.LastIndexByte(local, ':')
	if colon < 0 || colon == len(local)-1 {
		return 0, false, false
	}
	p, err := strconv.ParseUint(local[colon+1:], 16, 16)
	if err != nil {
		return 0, false, false
	}
	st, err := strconv.ParseUint(fields[3], 16, 8)
	if err != nil {
		return 0, false, false
	}
	return int(p), st == tcpListen, true
}
