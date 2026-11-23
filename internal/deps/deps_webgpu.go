//go:build deps && ((linux && cgo) || (js && wasm))

package deps

import _ "github.com/oliverbestmann/webgpu/wgpu"
