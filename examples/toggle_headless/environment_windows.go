package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

const usage = "commands: ls | create [message] | open [SESSION=…] | close | quit"

func newApplication() *gpui.Application {
	return platform.Application()
}

func displayEnvironment(arguments string) (gpui.GraphicalEnvironment, error) {
	environment := gpui.GraphicalEnvironmentDetect()
	for _, assignment := range strings.Fields(arguments) {
		session, ok := strings.CutPrefix(assignment, "SESSION=")
		if !ok {
			return gpui.GraphicalEnvironment{}, fmt.Errorf("expected SESSION=id, got %q", assignment)
		}
		id, err := strconv.ParseUint(session, 10, 32)
		if err != nil {
			return gpui.GraphicalEnvironment{}, fmt.Errorf("parsing session id: %w", err)
		}
		environment.SessionID = gpui.Some(uint32(id))
	}
	return environment, nil
}
