//go:build linux

package xrm

type howMatched uint8

const (
	howMatchedInstance howMatched = iota
	howMatchedClass
	howMatchedWildcard
)

type matchKind struct {
	skipped          bool
	precedingBinding binding
	howMatched       howMatched
}

func newMatch(precedingBinding binding, how howMatched) matchKind {
	return matchKind{precedingBinding: precedingBinding, howMatched: how}
}

type matchState struct {
	index   int
	history []matchKind
}

func (s matchState) skipLoose() matchState {
	history := append(append([]matchKind(nil), s.history...), matchKind{skipped: true})
	return matchState{index: s.index, history: history}
}

func (s matchState) step(kind matchKind) matchState {
	s.history = append(append([]matchKind(nil), s.history...), kind)
	s.index++
	return s
}

func checkMatch(e *entry, resource, class []string) [][]matchKind {
	states := []matchState{{}}
	n := max(len(resource), len(class))
	for i := range n {
		var res, cls *string
		if i < len(resource) {
			res = &resource[i]
		}
		if i < len(class) {
			cls = &class[i]
		}
		var next []matchState
		for _, state := range states {
			if state.index == len(e.components) {
				continue
			}
			b := e.components[state.index].binding
			if b == bindingLoose {
				next = append(next, state.skipLoose())
			}
			comp := e.components[state.index].component
			var kind matchKind
			matched := true
			switch {
			case comp.wildcard:
				kind = newMatch(b, howMatchedWildcard)
			case res != nil && comp.name == *res:
				kind = newMatch(b, howMatchedInstance)
			case cls != nil && comp.name == *cls:
				kind = newMatch(b, howMatchedClass)
			default:
				matched = false
			}
			if matched {
				next = append(next, state.step(kind))
			}
		}
		states = next
	}
	var result [][]matchKind
	for _, s := range states {
		if s.index == len(e.components) {
			result = append(result, s.history)
		}
	}
	return result
}

func rule1(m1, m2 matchKind) int {
	if !m1.skipped && m2.skipped {
		return 1
	}
	return 0
}

func rule2(m1, m2 matchKind) int {
	if !m1.skipped && m1.howMatched == howMatchedInstance && !m2.skipped && (m2.howMatched == howMatchedClass || m2.howMatched == howMatchedWildcard) {
		return 1
	}
	if !m1.skipped && m1.howMatched == howMatchedClass && !m2.skipped && m2.howMatched == howMatchedWildcard {
		return 1
	}
	return 0
}

func rule3(m1, m2 matchKind) int {
	if !m1.skipped && m1.precedingBinding == bindingTight && !m2.skipped && m2.precedingBinding == bindingLoose {
		return 1
	}
	return 0
}

func compareMatches(match1, match2 []matchKind) int {
	if len(match1) != len(match2) {
		panic("Both matches should have the same length (which is guaranteed by the current implementation of check_match())")
	}
	for i := range match1 {
		m1, m2 := match1[i], match2[i]
		for _, o := range [...]int{rule1(m1, m2), -rule1(m2, m1), rule2(m1, m2), -rule2(m2, m1), rule3(m1, m2), -rule3(m2, m1)} {
			if o != 0 {
				return o
			}
		}
	}
	return 0
}

func matchEntry(database []entry, resource, class string) ([]byte, bool) {
	res, ok := parseQuery([]byte(resource))
	if !ok {
		return nil, false
	}
	cls, ok := parseQuery([]byte(class))
	if !ok {
		return nil, false
	}
	var best *entry
	var bestMatch []matchKind
	for i := range database {
		e := &database[i]
		matches := checkMatch(e, res, cls)
		if len(matches) == 0 {
			continue
		}
		m := matches[0]
		for _, other := range matches[1:] {
			if compareMatches(other, m) >= 0 {
				m = other
			}
		}
		if best == nil || compareMatches(m, bestMatch) >= 0 {
			best, bestMatch = e, m
		}
	}
	if best == nil {
		return nil, false
	}
	return best.value, true
}
