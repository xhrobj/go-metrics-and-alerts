package sample

type Status string

// generate:reset
type ResettableState struct {
	Number     int
	Text       string
	Enabled    bool
	TextPtr    *string
	CounterPtr *int64
	Values     []int
	Labels     map[string]string
	Child      *NestedState
	Status     Status
	Codes      [3]int
}

type NestedState struct {
	Value int
}

func (s *NestedState) Reset() {
	s.Value = 0
}

type IgnoredState struct {
	Name string
}
