package checks

import (
	"context"
	"fmt"
	"strconv"
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

	all, aerr := d.Exec.run(ctx, path, "ps", "-a", "--format", "{{.Names}}|{{.State}}|{{.Status}}")
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
		fields := strings.Split(line, "|")
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimSpace(fields[0])
		st := strings.TrimSpace(fields[1])
		status := ""
		if len(fields) >= 3 {
			status = strings.TrimSpace(fields[2])
		}
		switch st {
		case "restarting", "dead":
			bad = append(bad, name+"("+st+")")
		case "exited":
			code, ok := exitedCode(status)
			if !ok || code != 0 {
				bad = append(bad, name+"("+st+")")
			}
		}
	}
	return bad
}

// exitedCode parses Docker Status like "Exited (137) 2 minutes ago".
// ok is false when the exit code cannot be extracted.
func exitedCode(status string) (int, bool) {
	const prefix = "Exited "
	i := strings.Index(status, prefix)
	if i < 0 {
		return 0, false
	}
	rest := status[i+len(prefix):]
	open := strings.IndexByte(rest, '(')
	if open < 0 {
		return 0, false
	}
	rest = rest[open+1:]
	close := strings.IndexByte(rest, ')')
	if close < 0 {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(rest[:close]))
	if err != nil {
		return 0, false
	}
	return n, true
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
