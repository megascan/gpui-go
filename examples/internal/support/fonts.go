// Package support holds the helpers the examples share (examples/example_support).
package support

import (
	"embed"
	"fmt"
	"os"
	"runtime"

	"github.com/megascan/gpui-go"
)

//go:embed fonts/*.ttf
var fontFiles embed.FS

var webFonts = []string{
	"IBMPlexSans-Regular.ttf",
	"IBMPlexSans-Italic.ttf",
	"IBMPlexSans-SemiBold.ttf",
	"IBMPlexSans-SemiBoldItalic.ttf",
	"Lilex-Regular.ttf",
	"Lilex-Bold.ttf",
	"Lilex-Italic.ttf",
	"Lilex-BoldItalic.ttf",
}

// Font returns the bytes of a bundled font file, e.g. "Lilex-Regular.ttf".
func Font(name string) []byte {
	data, err := fontFiles.ReadFile("fonts/" + name)
	if err != nil {
		panic(err)
	}
	return data
}

// LoadFonts loads the bundled application fonts on the web, where there are no system fonts;
// it reports whether the example can continue.
func LoadFonts(cx *gpui.App) bool {
	if runtime.GOOS != "js" {
		return true
	}
	fonts := make([][]byte, len(webFonts))
	for i, name := range webFonts {
		fonts[i] = Font(name)
	}
	if err := cx.TextSystem().AddFonts(fonts); err != nil {
		fmt.Fprintf(os.Stderr, "failed to load application fonts: %v\n", err)
		return false
	}
	return true
}
