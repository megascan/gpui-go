//go:build !race

package gpui

import "time"

func securityFV1Budget() time.Duration { return time.Second }
