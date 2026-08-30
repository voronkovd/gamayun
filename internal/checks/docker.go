package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/voronkovd/gamayun/internal/config"
)

type Docker struct {
	Cfg  config.Config
	Exec Exec
}

func (d Docker) Run(ctx context.Context) []Result {
	path, err := d.Exec.lookPath("docker")
	if err != nil {
		return []Result{
			skipped("docker.unhealthy", "docker not installed"),
			skipped("docker.state", "docker not installed"),
		}
	}

	var out []Result
	unhealthy, uerr := d.Exec.run(ctx, path, "ps", "--filter", "health=unhealthy", "--format", "{{.Names}}")
	if uerr != nil {
		out = append(out, fail("docker.unhealthy", "docker unhealthy: "+cliErr(uerr, unhealthy), nil))
	} else {
		names := nonEmptyLines(unhealthy)
		if len(names) > 0 {
			joined := strings.Join(names, ",")
			out = append(out, fail("docker.unhealthy", "docker unhealthy: "+joined, map[string]string{"names": joined}))
		} else {
			out = append(out, ok("docker.unhealthy", "docker: no unhealthy", map[string]string{"names": ""}))
		}
	}

	all, aerr := d.Exec.run(ctx, path, "ps", "-a", "--format", "{{.Names}} {{.State}}")
	if aerr != nil {
		out = append(out, fail("docker.state", "docker bad state: "+cliErr(aerr, all), nil))
	} else {
		bad := badDockerStates(all)
		if len(bad) > 0 {
			joined := strings.Join(bad, ",")
			out = append(out, fail("docker.state", "docker bad state: "+joined, map[string]string{"names": joined}))
		} else {
			out = append(out, ok("docker.state", "docker: no bad state", map[string]string{"names": ""}))
		}
	}

	if len(d.Cfg.Containers) == 0 {
		return out
	}

	runningOut, rerr := d.Exec.run(ctx, path, "ps", "--format", "{{.Names}}")
	running := map[string]bool{}
	if rerr == nil {
		for _, name := range nonEmptyLines(runningOut) {
			running[name] = true
		}
	}
	for _, name := range d.Cfg.Containers {
		key := "docker.required." + name
		if rerr != nil {
			out = append(out, fail(key, "docker: required container '"+name+"' not running", map[string]string{"running": "error"}))
			continue
		}
		if running[name] {
			out = append(out, ok(key, "docker: required container '"+name+"' running", map[string]string{"running": "yes"}))
			continue
		}
		out = append(out, fail(key, fmt.Sprintf("docker: required container '%s' not running", name), map[string]string{"running": "no"}))
	}
	return out
}

func badDockerStates(out string) []string {
	var bad []string
	for _, line := range nonEmptyLines(out) {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name, st := fields[0], fields[1]
		if st == "restarting" || st == "exited" || st == "dead" {
			bad = append(bad, name+"("+st+")")
		}
	}
	return bad
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func cliErr(err error, out string) string {
	if out != "" {
		return out
	}
	return err.Error()
}
