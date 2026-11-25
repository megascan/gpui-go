# M0-taffy-fixtures debt

- testdata/xml.zip is deflate-compressed with compress/flate BestCompression; byte-identical regeneration holds for one Go toolchain, a Go version that changes flate output changes the archive (regenerate and commit).
- xml_css_test.go is a minimal cssparser-like tokenizer (no escapes, strings, comments, urls); enough for the fixtures, not general CSS.
