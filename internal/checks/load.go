package checks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/voronkovd/gamayun/internal/config"
)

type Load struct {
	Cfg config.Config
}

func (l Load) Run(ctx context.Context) []Result {
	_ = ctx
	proc := l.Cfg.ProcFS
	if proc == "" {
		proc = "/proc"
	}
	load15, err := readLoad15(filepath.Join(proc, "loadavg"))
	if err != nil {
		return []Result{fail("load.15", "load15: cannot read /proc/loadavg", nil)}
	}
	metrics := map[string]string{
		"load15": formatFloat(load15),
		"max":    formatFloat(l.Cfg.Load15Max),
	}
	if load15 > l.Cfg.Load15Max {
		return []Result{fail("load.15", fmt.Sprintf("load15: %s (>%s)", formatFloat(load15), formatFloat(l.Cfg.Load15Max)), metrics)}
	}
	return []Result{ok("load.15", fmt.Sprintf("load15: %s", formatFloat(load15)), metrics)}
}

func readLoad15(path string) (float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return parseLoad15(string(data))
}

func parseLoad15(s string) (float64, error) {
	fields := strings.Fields(s)
	if len(fields) < 3 {
		return 0, fmt.Errorf("bad loadavg")
	}
	return strconv.ParseFloat(fields[2], 64)
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
