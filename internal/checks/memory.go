package checks

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/voronkovd/gamayun/internal/config"
)

type Memory struct {
	Cfg config.Config
}

func (m Memory) Run(ctx context.Context) []Result {
	_ = ctx
	proc := m.Cfg.ProcFS
	if proc == "" {
		proc = "/proc"
	}
	mb, err := readMemAvailable(filepath.Join(proc, "meminfo"))
	if err != nil {
		return []Result{fail("mem.available", "RAM: cannot read MemAvailable", nil)}
	}
	metrics := map[string]string{
		"mb":  strconv.Itoa(mb),
		"min": strconv.Itoa(m.Cfg.MemAvailMinMB),
	}
	if mb < m.Cfg.MemAvailMinMB {
		return []Result{fail("mem.available", fmt.Sprintf("RAM: only %dMB available (<%dMB)", mb, m.Cfg.MemAvailMinMB), metrics)}
	}
	return []Result{ok("mem.available", fmt.Sprintf("RAM: %dMB available", mb), metrics)}
}

func readMemAvailable(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return parseMemAvailable(f)
}

func parseMemAvailable(r io.Reader) (int, error) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "MemAvailable:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, fmt.Errorf("bad MemAvailable")
		}
		kb, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			return 0, err
		}
		return int(kb / 1024), nil
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("MemAvailable not found")
}
