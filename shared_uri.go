package gpui

// SharedURI is a SharedString containing a URI.
type SharedURI SharedString

// String returns the URI.
func (u SharedURI) String() string {
	return string(u)
}

// GoString formats the URI like its Rust Debug output.
func (u SharedURI) GoString() string {
	return SharedString(u).GoString()
}
