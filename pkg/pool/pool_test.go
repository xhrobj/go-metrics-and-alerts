package pool

import "testing"

type testObject struct {
	Number     int
	Text       string
	Values     []int
	ResetCalls int
}

func newTestObject() *testObject {
	return &testObject{
		Values: make([]int, 0, 4),
	}
}

func (o *testObject) Reset() {
	o.Number = 0
	o.Text = ""
	o.Values = o.Values[:0]
	o.ResetCalls++
}

func TestPoolGetUsesConstructor(t *testing.T) {
	pool := New(newTestObject)

	got := pool.Get()
	if got == nil {
		t.Fatal("Pool.Get() = nil, want object")
	}

	gotValuesCap := cap(got.Values)
	wantValuesCap := 4
	if gotValuesCap != wantValuesCap {
		t.Fatalf("cap(got.Values) = %d, want %d", gotValuesCap, wantValuesCap)
	}
}

func TestPoolPutResetsObject(t *testing.T) {
	pool := New(newTestObject)

	object := pool.Get()
	object.Number = 42
	object.Text = "value"
	object.Values = append(object.Values, 1, 2, 3)

	pool.Put(object)

	gotNumber := object.Number
	wantNumber := 0
	if gotNumber != wantNumber {
		t.Fatalf("object.Number = %d, want %d", gotNumber, wantNumber)
	}

	gotText := object.Text
	wantText := ""
	if gotText != wantText {
		t.Fatalf("object.Text = %q, want %q", gotText, wantText)
	}

	gotValuesLen := len(object.Values)
	wantValuesLen := 0
	if gotValuesLen != wantValuesLen {
		t.Fatalf("len(object.Values) = %d, want %d", gotValuesLen, wantValuesLen)
	}

	gotResetCalls := object.ResetCalls
	wantResetCalls := 1
	if gotResetCalls != wantResetCalls {
		t.Fatalf("object.ResetCalls = %d, want %d", gotResetCalls, wantResetCalls)
	}
}
