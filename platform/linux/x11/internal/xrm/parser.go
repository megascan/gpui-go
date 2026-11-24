//go:build linux

package xrm

func isOctalDigit(c byte) bool { return c >= '0' && c <= '7' }

func parseWithMatcher(data []byte, matcher func(byte) bool) ([]byte, []byte) {
	end := len(data)
	for i, c := range data {
		if !matcher(c) {
			end = i
			break
		}
	}
	return data[:end], data[end:]
}

func allowedInQuarkName(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_'
}

func nextComponent(data []byte) ([]byte, bool, []byte) {
	prefix, remaining := parseWithMatcher(data, allowedInQuarkName)
	if len(prefix) == 0 {
		return nil, false, remaining
	}
	return prefix, true, remaining
}

func skipToEOL(data []byte) []byte {
	_, rest := parseWithMatcher(data, func(c byte) bool { return c != '\n' })
	return rest
}

func skipSpaces(data []byte) []byte {
	_, rest := parseWithMatcher(data, func(c byte) bool { return c == ' ' })
	return rest
}

func skipText(data []byte, text string) ([]byte, bool) {
	if len(data) >= len(text) && string(data[:len(text)]) == text {
		return data[len(text):], true
	}
	return nil, false
}

func nextComponentName(data []byte) (component, bool, []byte) {
	if len(data) > 0 && data[0] == '?' {
		return component{wildcard: true}, true, data[1:]
	}
	comp, ok, remaining := nextComponent(data)
	if !ok {
		return component{}, false, remaining
	}
	return component{name: string(comp)}, true, remaining
}

func parseComponents(data []byte) ([]bound, []byte) {
	parseBinding := func(data []byte) (binding, []byte) {
		b := bindingTight
		for len(data) > 0 {
			switch data[0] {
			case '*':
				b = bindingLoose
			case '.':
			default:
				return b, data
			}
			data = data[1:]
		}
		return b, data
	}
	var result []bound
	for {
		b, remaining := parseBinding(data)
		comp, ok, remaining := nextComponentName(remaining)
		if !ok {
			break
		}
		data = remaining
		result = append(result, bound{binding: b, component: comp})
	}
	return result, data
}

func parseEntry(data []byte) (entry, bool, []byte) {
	components, data := parseComponents(data)
	if len(components) == 0 || components[len(components)-1].component.wildcard {
		return entry{}, false, skipToEOL(data)
	}
	data = skipSpaces(data)
	if len(data) == 0 || data[0] != ':' {
		return entry{}, false, skipToEOL(data)
	}
	data = data[1:]
	for {
		_, remaining := parseWithMatcher(data, func(c byte) bool { return c == ' ' || c == '\t' })
		if len(remaining) >= 2 && remaining[0] == '\\' && remaining[1] == '\n' {
			data = remaining[2:]
		} else {
			data = remaining
			break
		}
	}
	value := []byte{}
	index := 0
	var octal [2]byte
	octalLen := 0
	for index < len(data) {
		b := data[index]
		index++
		if b == '\n' {
			break
		}
		if octalLen > 0 {
			if isOctalDigit(b) {
				if octalLen == 1 {
					octal[1] = b
					octalLen = 2
				} else {
					x, y, z := octal[0]-'0', octal[1]-'0', b-'0'
					value = append(value, (x*8+y)*8+z)
					octalLen = 0
				}
				continue
			}
			value = append(value, '\\', octal[0])
			if octalLen == 2 {
				value = append(value, octal[1])
			}
			octalLen = 0
		}
		if b != '\\' {
			value = append(value, b)
			continue
		}
		if index >= len(data) {
			value = append(value, b)
			index--
		} else {
			switch x := data[index]; {
			case x == ' ':
				value = append(value, ' ')
			case x == '\t':
				value = append(value, '\t')
			case x == 'n':
				value = append(value, '\n')
			case x == '\\':
				value = append(value, '\\')
			case x == '\n':
			case isOctalDigit(x):
				octal[0] = x
				octalLen = 1
			default:
				value = append(value, b, x)
			}
		}
		index++
	}
	return entry{components: components, value: value}, true, data[index:]
}

func parseDatabase(data []byte, result *[]entry, includeCallback func(path []byte, entries *[]entry)) {
	for len(data) > 0 {
		switch data[0] {
		case '\n':
			data = data[1:]
		case '!':
			data = skipToEOL(data)
		case '#':
			remaining := skipSpaces(data[1:])
			data = skipToEOL(remaining)
			if remaining, ok := skipText(remaining, "include"); ok {
				_, remaining = parseWithMatcher(remaining, func(c byte) bool { return c == ' ' })
				if len(remaining) > 0 && remaining[0] == '"' {
					file, rest := parseWithMatcher(remaining[1:], func(c byte) bool { return c != '"' && c != '\n' })
					if len(rest) > 0 && rest[0] == '"' {
						includeCallback(file, result)
					}
				}
			}
		default:
			e, ok, remaining := parseEntry(data)
			data = remaining
			if ok {
				*result = append(*result, e)
			}
		}
	}
}

func parseQuery(data []byte) ([]string, bool) {
	var result []string
	for {
		comp, ok, remaining := nextComponent(data)
		if !ok {
			break
		}
		data = remaining
		for len(data) > 0 && data[0] == '.' {
			data = data[1:]
		}
		result = append(result, string(comp))
	}
	if len(data) == 0 {
		return result, true
	}
	return nil, false
}
