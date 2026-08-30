package checks

import (
	"context"
	"fmt"
	"strconv"
	"syscall"

	"github.com/voronkovd/gamayun/internal/config"
)

type Disk struct {
	Cfg  config.Config
	Path string
}

func (d Disk) Run(ctx context.Context) []Result {
	_ = ctx
	path := d.Path
	if path == "" {
		path = "/"
	}
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return []Result{fail("disk.root", "disk /: cannot read usage", nil)}
	}
	pct := usagePct(uint64(st.Blocks), uint64(st.Bfree), uint64(st.Bavail))
	metrics := map[string]string{
		"pct": strconv.Itoa(pct),
		"max": strconv.Itoa(d.Cfg.DiskPctMax),
	}
	if pct >= d.Cfg.DiskPctMax {
		return []Result{fail("disk.root", fmt.Sprintf("disk /: %d%% used (>=%d%%)", pct, d.Cfg.DiskPctMax), metrics)}
	}
	return []Result{ok("disk.root", fmt.Sprintf("disk /: %d%% used", pct), metrics)}
}

func usagePct(blocks, bfree, bavail uint64) int {
	if blocks < bfree {
		return 0
	}
	used := blocks - bfree
	den := used + bavail
	if den == 0 {
		return 0
	}
	return int(used * 100 / den)
}
