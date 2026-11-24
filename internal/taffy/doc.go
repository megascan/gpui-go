// Package taffy is a port of the taffy 0.15.0 layout engine with the features gpui enables:
// std, taffy_tree, flexbox, grid, block_layout, float_layout and content_size.
package taffy

//go:generate go run ./internal/mkfixtures -src ../../_ext/taffy/tests/xml -out testdata/xml.zip
