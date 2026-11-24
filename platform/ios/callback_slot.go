//go:build darwin

package ios

type callbackSlot[T any] struct {
	value      T
	present    bool
	generation uint64
}

func (s *callbackSlot[T]) set(value T) {
	s.generation++
	s.value = value
	s.present = true
}

func (s *callbackSlot[T]) take() (T, bool) {
	s.generation++
	value, ok := s.value, s.present
	var zero T
	s.value, s.present = zero, false
	return value, ok
}

func (s *callbackSlot[T]) with(callback func(value *T)) bool {
	_, ok := callbackSlotWithR(s, func(value *T) struct{} {
		callback(value)
		return struct{}{}
	})
	return ok
}

func callbackSlotWithR[T, R any](s *callbackSlot[T], callback func(value *T) R) (R, bool) {
	if !s.present {
		var zero R
		return zero, false
	}
	value := s.value
	var zero T
	s.value, s.present = zero, false
	generation := s.generation
	result := callback(&value)
	if s.generation == generation {
		s.value, s.present = value, true
	}
	return result, true
}
