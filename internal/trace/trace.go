// Package trace runs the system traceroute and streams its hops.
package trace

import (
	"bufio"
	"context"
	"os/exec"
	"strconv"
	"strings"
)

// Hop is one traceroute line. IP is empty when the node did not answer.
type Hop struct {
	N  int
	IP string
	MS float64
}

// ParseHop reads a line of `traceroute -n -q 1` output; ok is false for headers and noise.
func ParseHop(line string) (h Hop, ok bool) {
	f := strings.Fields(line)
	n, err := strconv.Atoi(firstOr(f, ""))
	if err != nil || len(f) < 2 {
		return h, false
	}
	h.N = n
	if f[1] == "*" {
		return h, true
	}
	h.IP = f[1]
	if len(f) > 2 {
		h.MS, _ = strconv.ParseFloat(f[2], 64)
	}
	return h, true
}

func firstOr(f []string, def string) string {
	if len(f) == 0 {
		return def
	}
	return f[0]
}

// Run calls cb for every hop as it arrives. host must not start with "-" (checked by the caller; "--" is passed anyway).
func Run(ctx context.Context, host string, cb func(Hop)) error {
	cmd := exec.CommandContext(ctx, "traceroute", "-n", "-q", "1", "-w", "2", "-m", "20", "--", host)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	sc := bufio.NewScanner(out)
	for sc.Scan() {
		if h, ok := ParseHop(sc.Text()); ok {
			cb(h)
		}
	}
	return cmd.Wait()
}
