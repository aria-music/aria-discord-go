package aria

import (
	"reflect"
	"testing"
)

func TestStoreStateInitialValue(t *testing.T) {
	s := store{}
	var want *stateData
	got := s.getState()
	t.Logf("type want: %v, type got: %v\n", reflect.TypeOf(want), reflect.TypeOf(got))
	if got != want {
		t.Errorf("want: %v, got: %v\n", want, got)
	}
}

func TestStoreStateSet(t *testing.T) {
	s := store{}
	want := &stateData{}
	s.setState(want)
	if s.state != want {
		t.Errorf("want: %p, got: %p\n", want, s.state)
	}
}

func TestStoreStateSetValue(t *testing.T) {
	s := store{}
	want := &stateData{
		State: "TestState",
	}
	s.setState(want)
	if s.state.State != want.State {
		t.Errorf("want: %s, got: %s\n", want.State, s.state.State)
	}
}

func TestStoreStateGet(t *testing.T) {
	s := store{
		state: &stateData{},
	}
	got := s.getState()
	if s.state != got {
		t.Errorf("want: %p, got: %p\n", s.state, got)
	}
}

func TestStoreStateGetValue(t *testing.T) {
	s := store{
		state: &stateData{
			State: "TestState",
		},
	}
	got := s.getState()
	if s.state.State != got.State {
		t.Errorf("want: %s, got: %s\n", s.state.State, got.State)
	}
}
