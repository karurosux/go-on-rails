package run

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// CheckPortFree reports an error when TCP port is already taken, naming the
// owning process when it can. `gor dev` refuses to start on a taken port:
// otherwise Air's server dies on the bind while Vite happily proxies /api to
// whatever stale process is squatting there, and every new route 404s with no
// hint why.
func CheckPortFree(port string) error {
	l, err := net.Listen("tcp", ":"+port)
	if err == nil {
		_ = l.Close()
		return nil
	}
	msg := fmt.Sprintf("port %s is already in use", port)
	if owner := portOwner(port); owner != "" {
		msg += " by " + owner
	}
	return fmt.Errorf("%s.\nStop that process (or change PORT in .env) and run `gor dev` again", msg)
}

// portOwner best-effort identifies what listens on port, e.g.
// "`go run ./cmd/server` (pid 26362, started Sun Oct 18 20:50)".
func portOwner(port string) string {
	out, err := exec.Command("lsof", "-nP", "-iTCP:"+port, "-sTCP:LISTEN", "-Fpc").Output()
	if err != nil {
		return ""
	}
	// -F output: lines like "p26362" (pid) and "cserver" (command).
	var pid, command string
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "p") && pid == "":
			pid = line[1:]
		case strings.HasPrefix(line, "c") && command == "":
			command = line[1:]
		}
	}
	if pid == "" {
		return ""
	}
	desc := fmt.Sprintf("pid %s", pid)
	if started := processStart(pid); started != "" {
		desc += ", started " + started
	}
	if command != "" {
		return fmt.Sprintf("`%s` (%s)", command, desc)
	}
	return "(" + desc + ")"
}

func processStart(pid string) string {
	out, err := exec.Command("ps", "-o", "lstart=", "-p", pid).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// WatchPort waits for something to start listening on port, and calls onDead
// once if nothing does within timeout. `gor dev` uses it as a watchdog: Air
// stays alive even when the server it launched exited at startup, so without
// this a dev session can look healthy while /api has nothing behind it.
func WatchPort(port string, timeout time.Duration, onDead func()) {
	go func() {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, time.Second)
			if err == nil {
				_ = conn.Close()
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		onDead()
	}()
}
