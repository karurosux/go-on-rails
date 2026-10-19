// Package run executes multiple long-lived processes with prefixed, colored
// output (used by `gor dev`), and single commands with inherited output.
package run

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	colorReset   = "\033[0m"
	ColorCyan    = "\033[36m"
	ColorMagenta = "\033[35m"
	ColorYellow  = "\033[33m"
)

// Proc is one process managed by Procs.
type Proc struct {
	Name  string // output prefix, e.g. "go"
	Color string
	Dir   string
	Env   []string // extra env vars, KEY=VALUE
	Cmd   string
	Args  []string
}

type running struct {
	proc Proc
	cmd  *exec.Cmd
}

// Procs starts every proc, prefixes each output line with its colored name,
// and shuts everything down cleanly when the user hits Ctrl+C or any proc
// exits on its own.
func Procs(procs []Proc) error {
	var mu sync.Mutex // serializes output lines
	var wg sync.WaitGroup
	started := make([]running, 0, len(procs))
	done := make(chan string, len(procs))

	prefix := func(p Proc, line string) {
		mu.Lock()
		defer mu.Unlock()
		fmt.Printf("%s[%s]%s %s\n", p.Color, p.Name, colorReset, line)
	}

	for _, p := range procs {
		cmd := exec.Command(p.Cmd, p.Args...)
		cmd.Dir = p.Dir
		cmd.Env = append(os.Environ(), p.Env...)
		setProcAttrs(cmd) // own process group: stopping it takes its children too
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		cmd.Stderr = cmd.Stdout // same pipe: stderr lines get the same prefix
		if err := cmd.Start(); err != nil {
			stopAll(started)
			return fmt.Errorf("start %s (%s): %w", p.Name, p.Cmd, err)
		}
		r := running{p, cmd}
		started = append(started, r)

		wg.Add(1)
		go func(p Proc, cmd *exec.Cmd) {
			defer wg.Done()
			sc := bufio.NewScanner(stdout)
			sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for sc.Scan() {
				prefix(p, sc.Text())
			}
			_ = cmd.Wait()
			done <- p.Name
		}(p, cmd)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)

	select {
	case <-sig:
		fmt.Printf("\n%s[gor]%s shutting down...\n", ColorYellow, colorReset)
	case name := <-done:
		fmt.Printf("%s[gor]%s %s exited, stopping everything\n", ColorYellow, colorReset, name)
	}

	stopAll(started)
	wg.Wait()
	return nil
}

// stopAll interrupts every process group, then kills any that have not
// exited after a grace period. Signals go to the group, not just the direct
// child, so wrappers like air and npm cannot leave their real servers behind.
func stopAll(started []running) {
	for _, r := range started {
		interruptProc(r.cmd)
	}
	deadline := time.After(5 * time.Second)
	for _, r := range started {
		exited := make(chan struct{})
		go func(cmd *exec.Cmd) {
			for cmd.ProcessState == nil {
				time.Sleep(50 * time.Millisecond)
			}
			close(exited)
		}(r.cmd)
		select {
		case <-exited:
		case <-deadline:
			killProc(r.cmd)
		}
		// The group may have stragglers the direct child did not reap.
		killProc(r.cmd)
	}
}

// Command runs a single command in dir, streaming output to the terminal.
func Command(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w", name, args, err)
	}
	return nil
}

// CommandQuiet runs a command and returns its combined output only on error.
func CommandQuiet(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v failed: %w\n%s", name, args, err, out)
	}
	return nil
}
