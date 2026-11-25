//go:build (!unix && !windows) || android

package gotext

import "github.com/go-text/typesetting/language"

var platformCommonFallback = []string{}

var platformForbiddenFallback = []string{}

func scriptFallback(script language.Script, locale string) []string {
	return nil
}
