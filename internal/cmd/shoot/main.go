// Command shoot builds an example, runs it, captures only its window with grim and terminates it.
package main

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type client struct {
	Address  string `json:"address"`
	Mapped   bool   `json:"mapped"`
	Hidden   bool   `json:"hidden"`
	At       [2]int `json:"at"`
	Size     [2]int `json:"size"`
	Pid      int    `json:"pid"`
	Xwayland bool   `json:"xwayland"`
}

type options struct {
	pkg      string
	out      string
	x11      bool
	mayExit  bool
	noWindow time.Duration
	timeout  time.Duration
	settle   time.Duration
	size     string
	args     []string
}

func main() {
	var o options
	flag.StringVar(&o.pkg, "pkg", "", "package to build and run, e.g. ./examples/hello_world")
	flag.StringVar(&o.out, "out", "", "PNG path for the window capture")
	flag.BoolVar(&o.x11, "x11", false, "run with WAYLAND_DISPLAY unset (XWayland)")
	flag.BoolVar(&o.mayExit, "may-exit", false, "accept a program that exits 0 before a window appears")
	flag.DurationVar(&o.noWindow, "no-window", 0, "the program never opens a window; succeed if it is alive at this deadline or exited 0")
	flag.DurationVar(&o.timeout, "timeout", 30*time.Second, "time to wait for the window")
	flag.DurationVar(&o.settle, "settle", 1500*time.Millisecond, "delay between the window mapping and the capture")
	flag.StringVar(&o.size, "size", "", "float the window and resize it to WxH logical pixels instead of the compositor's tiled size")
	flag.Parse()
	o.args = flag.Args()
	if o.pkg == "" || (o.out == "" && o.noWindow == 0) {
		fmt.Fprintln(os.Stderr, "usage: shoot -pkg <package> -out <file.png> [-x11] [-may-exit] [-no-window <duration>] [-- args]")
		os.Exit(2)
	}
	if err := run(o); err != nil {
		fmt.Fprintln(os.Stderr, "shoot:", err)
		os.Exit(1)
	}
}

func run(o options) error {
	dir, err := os.MkdirTemp("", "gpui-shoot-")
	if err != nil {
		return fmt.Errorf("creating build directory: %w", err)
	}
	defer os.RemoveAll(dir)
	bin := filepath.Join(dir, "example")
	build := exec.Command("go", "build", "-o", bin, o.pkg)
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("building %s: %w", o.pkg, err)
	}

	before := map[string]bool{}
	if o.noWindow == 0 {
		existing, err := clients()
		if err != nil {
			return err
		}
		for _, c := range existing {
			before[c.Address] = true
		}
	}

	cmd := exec.Command(bin, o.args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	cmd.Env = os.Environ()
	if o.x11 {
		cmd.Env = withoutEnv(cmd.Env, "WAYLAND_DISPLAY")
		if os.Getenv("DISPLAY") == "" {
			return errors.New("-x11 needs DISPLAY")
		}
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", o.pkg, err)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	done := false
	defer func() {
		if !done {
			terminate(cmd, exited)
		}
	}()

	if o.noWindow > 0 {
		select {
		case err := <-exited:
			done = true
			if err != nil {
				return fmt.Errorf("program failed: %w", err)
			}
			return nil
		case <-time.After(o.noWindow):
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
	defer cancel()
	var win client
	for {
		select {
		case err := <-exited:
			done = true
			if err == nil && o.mayExit {
				fmt.Fprintln(os.Stderr, "shoot: program exited 0 before opening a window")
				return nil
			}
			return fmt.Errorf("program exited before opening a window: %v", err)
		case <-ctx.Done():
			return fmt.Errorf("no window from pid %d within %s", cmd.Process.Pid, o.timeout)
		case <-time.After(100 * time.Millisecond):
		}
		found, ok, err := findWindow(cmd.Process.Pid, before, o.x11)
		if err != nil {
			return err
		}
		if ok {
			win = found
			break
		}
	}
	if o.size != "" {
		if err := floatAndResize(win.Address, o.size); err != nil {
			return err
		}
	}

	select {
	case err := <-exited:
		done = true
		return fmt.Errorf("program exited after opening its window: %v", err)
	case <-time.After(o.settle):
	}
	latest, ok, err := findWindow(cmd.Process.Pid, before, o.x11)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("window disappeared before capture")
	}
	win = latest
	if err := os.MkdirAll(filepath.Dir(o.out), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	geometry := fmt.Sprintf("%d,%d %dx%d", win.At[0], win.At[1], win.Size[0], win.Size[1])
	grim := exec.Command("grim", "-g", geometry, o.out)
	grim.Stdout = os.Stderr
	grim.Stderr = os.Stderr
	if err := grim.Run(); err != nil {
		return fmt.Errorf("grim: %w", err)
	}
	fmt.Fprintf(os.Stderr, "shoot: captured %s (%s)\n", o.out, geometry)
	return nil
}

func findWindow(pid int, before map[string]bool, x11 bool) (client, bool, error) {
	all, err := clients()
	if err != nil {
		return client{}, false, err
	}
	for _, c := range all {
		if !c.Mapped || c.Hidden || c.Size[0] == 0 || c.Size[1] == 0 {
			continue
		}
		if c.Pid == pid {
			return c, true, nil
		}
	}
	if x11 {
		var found []client
		for _, c := range all {
			if c.Mapped && !c.Hidden && c.Xwayland && !before[c.Address] && c.Size[0] > 0 && c.Size[1] > 0 && inProcessGroup(c.Pid, pid) {
				found = append(found, c)
			}
		}
		if len(found) > 1 {
			return client{}, false, fmt.Errorf("%d new XWayland windows belong to process group %d, refusing to guess", len(found), pid)
		}
		if len(found) == 1 {
			return found[0], true, nil
		}
	}
	return client{}, false, nil
}

func inProcessGroup(pid, pgid int) bool {
	if pid <= 0 {
		return false
	}
	got, err := syscall.Getpgid(pid)
	return err == nil && got == pgid
}

func clients() ([]client, error) {
	out, err := exec.Command("hyprctl", "clients", "-j").Output()
	if err != nil {
		return nil, fmt.Errorf("hyprctl clients: %w", err)
	}
	var cs []client
	if err := json.Unmarshal(out, &cs); err != nil {
		return nil, fmt.Errorf("parsing hyprctl clients: %w", err)
	}
	return cs, nil
}

func terminate(cmd *exec.Cmd, exited <-chan error) {
	pgid := -cmd.Process.Pid
	_ = syscall.Kill(pgid, syscall.SIGTERM)
	select {
	case <-exited:
	case <-time.After(3 * time.Second):
		_ = syscall.Kill(pgid, syscall.SIGKILL)
		<-exited
	}
}

func withoutEnv(env []string, key string) []string {
	prefix := key + "="
	out := env[:0:0]
	for _, kv := range env {
		if len(kv) >= len(prefix) && kv[:len(prefix)] == prefix {
			continue
		}
		out = append(out, kv)
	}
	return out
}

func floatAndResize(address, size string) error {
	var w, h int
	if _, err := fmt.Sscanf(size, "%dx%d", &w, &h); err != nil || w <= 0 || h <= 0 {
		return fmt.Errorf("invalid -size %q, want WxH", size)
	}
	target := "address:" + address
	for _, dispatch := range []string{
		fmt.Sprintf(`hl.dsp.window.float({ action = "enable", window = %q })`, target),
		fmt.Sprintf(`hl.dsp.window.resize({ x = %d, y = %d, relative = false, window = %q })`, w, h, target),
	} {
		out, err := exec.Command("hyprctl", "dispatch", dispatch).CombinedOutput()
		if err != nil || !bytes.HasPrefix(bytes.TrimSpace(out), []byte("ok")) {
			return fmt.Errorf("hyprctl dispatch %s: %s: %v", dispatch, bytes.TrimSpace(out), err)
		}
	}
	return nil
}
