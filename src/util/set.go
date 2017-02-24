package util

type Set struct {
	data map[interface{}]struct{}
}

func NewSet(s ...interface{}) *Set {
	set := Set{
		data: make(map[interface{}]struct{}),
	}

	for _, item := range s {
		set.Add(item)
	}
	return &set
}

func (s *Set) Add(v interface{}) bool {
	_, found := s.data[v]
	s.data[v] = struct{}{}
	return !found //False if it existed already
}

func (s *Set) Remove(v interface{}) bool {
	delete(s.data, v)
	return true
}

func (s *Set) Exist(v interface{}) bool {
	_, found := s.data[v]
	return found
}

func (s *Set) Size() int {
	return len(s.data)
}
