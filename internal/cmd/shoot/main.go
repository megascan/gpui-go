//go:build !windows && !js

// Command shoot builds an example, runs it, captures only its window with grim and terminates it.
//
// The windows never appear on the owner's workspaces: a Hyprland Lua hook (`hyprctl eval`,
// window.open_early) moves every window of the child's process group, unfocused and floating, to a
// hidden special workspace before it maps, and grim captures the toplevel itself (`grim -T`), so
// monitor layout and the owner's windows do not matter.
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
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type workspace struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type client struct {
	Address   string    `json:"address"`
	Mapped    bool      `json:"mapped"`
	At        [2]int    `json:"at"`
	Size      [2]int    `json:"size"`
	Pid       int       `json:"pid"`
	Xwayland  bool      `json:"xwayland"`
	Floating  bool      `json:"floating"`
	Workspace workspace `json:"workspace"`
	StableID  string    `json:"stableId"`
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
	flag.DurationVar(&o.settle, "settle", 1500*time.Millisecond, "delay between the window reaching its size and the capture")
	flag.StringVar(&o.size, "size", "800x600", "logical WxH every window of the program is floated and resized to")
	flag.Parse()
	o.args = flag.Args()
	if o.pkg == "" || (o.out == "" && o.noWindow == 0) {
		fmt.Fprintln(os.Stderr, "usage: shoot -pkg <package> -out <file.png> [-x11] [-may-exit] [-size WxH] [-no-window <duration>] [-- args]")
		os.Exit(2)
	}
	if err := run(o); err != nil {
		fmt.Fprintln(os.Stderr, "shoot:", err)
		os.Exit(1)
	}
}

type session struct {
	cmd      *exec.Cmd
	exited   chan error
	done     bool
	cleanups []func()
}

func (s *session) cleanup() {
	if s.cmd != nil && s.cmd.Process != nil && !s.done {
		terminate(s.cmd, s.exited)
		s.done = true
	}
	for i := len(s.cleanups) - 1; i >= 0; i-- {
		s.cleanups[i]()
	}
	s.cleanups = nil
}

func run(o options) error {
	var w, h int
	if o.size != "" {
		if _, err := fmt.Sscanf(o.size, "%dx%d", &w, &h); err != nil || w <= 0 || h <= 0 {
			return fmt.Errorf("invalid -size %q, want WxH", o.size)
		}
	}
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

	s := &session{}
	defer s.cleanup()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigs)
	interrupted := make(chan struct{})
	go func() {
		if _, ok := <-sigs; ok {
			close(interrupted)
		}
	}()

	gateR, gateW, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("creating start gate: %w", err)
	}
	defer gateW.Close()
	shArgs := append([]string{"-c", `read -r _ <&3; exec 3<&-; exec "$0" "$@"`, bin}, o.args...)
	cmd := exec.Command("/bin/sh", shArgs...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{gateR}
	cmd.SysProcAttr = childProcAttr()
	cmd.Env = os.Environ()
	if o.x11 {
		cmd.Env = withoutEnv(cmd.Env, "WAYLAND_DISPLAY")
		if os.Getenv("DISPLAY") == "" {
			return errors.New("-x11 needs DISPLAY")
		}
	}
	if err := cmd.Start(); err != nil {
		gateR.Close()
		return fmt.Errorf("starting %s: %w", o.pkg, err)
	}
	gateR.Close()
	s.cmd = cmd
	s.exited = make(chan error, 1)
	go func() { s.exited <- cmd.Wait() }()
	pid := cmd.Process.Pid

	key := fmt.Sprintf("gpui-shoot-%d", pid)
	special := "special:" + key
	if err := installHook(key, special, pid, w, h, o.x11); err != nil {
		return err
	}
	s.cleanups = append(s.cleanups, func() { removeHook(key) })
	if _, err := gateW.Write([]byte("go\n")); err != nil {
		return fmt.Errorf("releasing start gate: %w", err)
	}
	gateW.Close()

	if o.noWindow > 0 {
		select {
		case err := <-s.exited:
			s.done = true
			if err != nil {
				return fmt.Errorf("program failed: %w", err)
			}
			return nil
		case <-interrupted:
			return errors.New("interrupted")
		case <-time.After(o.noWindow):
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
	defer cancel()
	var win client
	stable, sized := 0, 0
	for {
		select {
		case err := <-s.exited:
			s.done = true
			if win.Address == "" && err == nil && o.mayExit {
				fmt.Fprintln(os.Stderr, "shoot: program exited 0 before opening a window")
				return nil
			}
			if win.Address == "" {
				return fmt.Errorf("program exited before opening a window: %v", err)
			}
			return fmt.Errorf("program exited after opening its window: %v", err)
		case <-interrupted:
			return errors.New("interrupted")
		case <-ctx.Done():
			if win.Address == "" {
				return fmt.Errorf("no window from pid %d within %s", pid, o.timeout)
			}
			return fmt.Errorf("window %s did not settle on %s within %s (last %dx%d on %q)", win.Address, special, o.timeout, win.Size[0], win.Size[1], win.Workspace.Name)
		case <-time.After(100 * time.Millisecond):
		}
		found, ok, err := findWindow(pid, o.x11)
		if err != nil {
			return err
		}
		if !ok {
			if win.Address != "" {
				return errors.New("window disappeared before capture")
			}
			continue
		}
		if found.Workspace.Name != special {
			return fmt.Errorf("window %s mapped on workspace %q instead of %s; the Hyprland hook missed it", found.Address, found.Workspace.Name, special)
		}
		if found.Address == win.Address && found.Size == win.Size && found.At == win.At && found.Floating {
			stable++
		} else {
			stable = 0
		}
		win = found
		if stable < 3 {
			continue
		}
		if w == 0 || win.Size == [2]int{w, h} {
			break
		}
		if sized++; sized >= 20 {
			fmt.Fprintf(os.Stderr, "shoot: window kept %dx%d instead of -size %dx%d\n", win.Size[0], win.Size[1], w, h)
			break
		}
	}

	select {
	case err := <-s.exited:
		s.done = true
		return fmt.Errorf("program exited after opening its window: %v", err)
	case <-interrupted:
		return errors.New("interrupted")
	case <-time.After(o.settle):
	}
	latest, ok, err := findWindow(pid, o.x11)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("window disappeared before capture")
	}
	if latest.Workspace.Name != special {
		return fmt.Errorf("window left %s before capture (now on %q)", special, latest.Workspace.Name)
	}
	if latest.StableID == "" {
		return errors.New("hyprctl reports no stableId for the window; grim -T needs it")
	}
	if err := os.MkdirAll(filepath.Dir(o.out), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	grim := exec.Command("grim", "-T", latest.StableID, o.out)
	grim.Stdout = os.Stderr
	grim.Stderr = os.Stderr
	if err := grim.Run(); err != nil {
		return fmt.Errorf("grim -T %s: %w", latest.StableID, err)
	}
	fmt.Fprintf(os.Stderr, "shoot: captured %s (toplevel %s, %dx%d logical)\n", o.out, latest.StableID, latest.Size[0], latest.Size[1])
	return nil
}

func installHook(key, special string, pid, w, h int, x11 bool) error {
	lua := fmt.Sprintf(`
local P, X11, KEY, SPECIAL, W, H = %d, %t, %q, %q, %d, %d
local function alive(pid)
  local f = io.open("/proc/" .. pid .. "/stat", "r")
  if not f then return false end
  f:close()
  return true
end
_G.__gpui_shoot = _G.__gpui_shoot or {}
for k, t in pairs(_G.__gpui_shoot) do
  if not alive(t.pid) then
    t.sub:remove()
    _G.__gpui_shoot[k] = nil
  end
end
if not _G.__gpui_shoot_rule then
  _G.__gpui_shoot_rule = hl.window_rule({
    name = "gpui-shoot",
    match = { tag = "gpui-shoot" },
    no_anim = true,
    render_unfocused = true,
  })
end
local function ours(pid)
  if pid == P then return true end
  if not X11 or type(pid) ~= "number" or pid <= 0 then return false end
  local f = io.open("/proc/" .. pid .. "/stat", "r")
  if not f then return false end
  local line = f:read("*l")
  f:close()
  local pgrp = line and line:match("%%)%%s+%%S+%%s+%%S+%%s+(%%d+)")
  return tonumber(pgrp) == P
end
local sub = hl.on("window.open_early", function(w)
  if not (w and ours(w.pid)) then return end
  local target = "address:" .. w.address
  hl.dispatch(hl.dsp.window.set_prop({ prop = "no_focus", value = "1", window = target }))
  hl.dispatch(hl.dsp.window.tag({ tag = "+gpui-shoot", window = target }))
  hl.dispatch(hl.dsp.window.move({ workspace = SPECIAL, follow = false, window = target }))
  hl.dispatch(hl.dsp.window.float({ action = "enable", window = target }))
  if W > 0 and H > 0 then
    hl.dispatch(hl.dsp.window.resize({ x = W, y = H, relative = false, window = target }))
  end
  hl.dispatch(hl.dsp.window.center({ window = target }))
end)
_G.__gpui_shoot[KEY] = { pid = P, sub = sub }
`, pid, x11, key, special, w, h)
	return hyprEval(lua)
}

func removeHook(key string) {
	lua := fmt.Sprintf(`
local all = _G.__gpui_shoot
local t = all and all[%q]
if t then
  t.sub:remove()
  all[%q] = nil
end
if _G.__gpui_shoot_rule and (not all or next(all) == nil) then
  _G.__gpui_shoot_rule:set_enabled(false)
  _G.__gpui_shoot_rule = nil
end
`, key, key)
	if err := hyprEval(lua); err != nil {
		fmt.Fprintln(os.Stderr, "shoot: removing Hyprland hook:", err)
	}
}

func hyprEval(lua string) error {
	out, err := exec.Command("hyprctl", "eval", lua).CombinedOutput()
	if err != nil || !bytes.HasPrefix(bytes.TrimSpace(out), []byte("ok")) {
		return fmt.Errorf("hyprctl eval: %s: %v", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func findWindow(pid int, x11 bool) (client, bool, error) {
	all, err := clients()
	if err != nil {
		return client{}, false, err
	}
	for _, c := range all {
		if !c.Mapped || c.Size[0] == 0 || c.Size[1] == 0 {
			continue
		}
		if c.Pid == pid || (x11 && c.Xwayland && inProcessGroup(c.Pid, pid)) {
			return c, true, nil
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
		if strings.HasPrefix(kv, prefix) {
			continue
		}
		out = append(out, kv)
	}
	return out
}
