//go:build linux

package xcbshim

const (
	setupSize       uintptr = 40
	formatSize      uintptr = 8
	screenSize      uintptr = 40
	depthSize       uintptr = 8
	visualTypeSize  uintptr = 24
	setupVendorLen  uintptr = 24
	setupRootsLen   uintptr = 28
	setupFormatsLen uintptr = 29
	screenRootVis   uintptr = 32
	screenDepthsLen uintptr = 39
	depthVisualsLen uintptr = 2
	depthDepth      uintptr = 0
	visualTypeID    uintptr = 0
)

func screenAt(setup uintptr, index int) (uintptr, bool) {
	if setup == 0 || index < 0 {
		return 0, false
	}
	vendor := uintptr(load16(setup + setupVendorLen))
	p := setup + setupSize + vendor
	p = (p + 3) &^ 3
	p += uintptr(load8(setup+setupFormatsLen)) * formatSize
	roots := int(load8(setup + setupRootsLen))
	if index >= roots {
		return 0, false
	}
	for i := 0; i < index; i++ {
		p = nextScreen(p)
	}
	return p, true
}

func nextScreen(screen uintptr) uintptr {
	p := screen + screenSize
	depths := int(load8(screen + screenDepthsLen))
	for range depths {
		visuals := uintptr(load16(p + depthVisualsLen))
		p += depthSize + visuals*visualTypeSize
	}
	return p
}

func rootVisualOf(setup uintptr, index int) (uint32, bool) {
	screen, ok := screenAt(setup, index)
	if !ok {
		return 0, false
	}
	return load32(screen + screenRootVis), true
}

func visualForDepthOf(setup uintptr, index int, depth uint8) (uint32, bool) {
	screen, ok := screenAt(setup, index)
	if !ok {
		return 0, false
	}
	p := screen + screenSize
	for range int(load8(screen + screenDepthsLen)) {
		visuals := int(load16(p + depthVisualsLen))
		if load8(p+depthDepth) == depth {
			for v := range visuals {
				if id := load32(p + depthSize + uintptr(v)*visualTypeSize + visualTypeID); id != 0 {
					return id, true
				}
			}
		}
		p += depthSize + uintptr(visuals)*visualTypeSize
	}
	return 0, false
}
