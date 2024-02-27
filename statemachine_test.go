package sm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func newRandomIDHandler(f HandleTransition) Handler {
	return Handler{ID: uuid.NewString(), H: f}
}

func TestNewStateMachine(t *testing.T) {
	tests := []struct {
		name        string
		transitions []Transition
		current     string
		want        *StateMachine
		wantPanic   bool
	}{
		{
			name:        "empty",
			transitions: []Transition{},
			current:     "foo",
			wantPanic:   true,
		},
		{
			name: "valid",
			transitions: []Transition{
				NewTransition("foo", "bar", Handler{}),
				NewTransition("bar", "baz", Handler{}),
			},
			current: "foo",
			want: &StateMachine{
				name: "TestNewStateMachine",
				transitions: map[string]map[string]Handler{
					"foo": {
						"bar": Handler{},
					},
					"bar": {
						"baz": Handler{},
					},
				},
				current: "foo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); (r != nil) != tt.wantPanic {
					t.Errorf("NewStateMachine() recover = %v, wantPanic %v", r, tt.wantPanic)
				}
			}()

			got := NewStateMachine("TestNewStateMachine", tt.transitions, tt.current)
			if !got.Equals(tt.want) {
				t.Errorf("NewStateMachine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStateMachine_Transition(t *testing.T) {
	sm := NewStateMachine("TestStateMachine_Transition", []Transition{
		NewTransition("foo", "bar", Handler{}),
		NewTransition("foo", "bax", Handler{H: func(from, to string, payload any) string {
			return "foo"
		}}),
		NewTransition("bar", "baz", Handler{}),
	}, "foo")

	tests := []struct {
		name        string
		next        string
		wantCurrent string
		wantError   bool
	}{
		{
			name:        "valid",
			next:        "bar",
			wantCurrent: "bar",
			wantError:   false,
		},
		{
			name:      "invalid",
			next:      "baz",
			wantError: true,
		},
		{
			name:        "unexpected",
			next:        "bax",
			wantCurrent: "foo",
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sm.Clone().Transition(tt.next, nil)
			if (err != nil) != tt.wantError {
				t.Errorf("StateMachine.Transition() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.wantError {
				t.Logf("error = %v", err)
			}
		})
	}
}

func TestStateMachine_CanTransition(t *testing.T) {
	sm := NewStateMachine("TestStateMachine_CanTransition", []Transition{
		NewTransition("foo", "bar", Handler{}),
		NewTransition("foo", "bax", Handler{}),
		NewTransition("bar", "baz", Handler{}),
	}, "foo")

	tests := []struct {
		name string
		next string
		want bool
	}{
		{
			name: "valid",
			next: "bar",
			want: true,
		},
		{
			name: "invalid",
			next: "bax",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sm.CanTransition(tt.next)
			if got != tt.want {
				t.Errorf("StateMachine.CanTransition() = %v, want %v", got, tt.want)
			}
			sm.Transition(tt.next, nil)
		})
	}
}

func TestStateMachine_Transitions(t *testing.T) {
	sm := NewStateMachine("TestStateMachine_Transitions", []Transition{
		NewTransition("foo", "bar", Handler{}),
		NewTransition("foo", "bax", Handler{}),
		NewTransition("bar", "baz", Handler{}),
	}, "foo")

	got := sm.Transitions()
	want := []string{"bar", "bax"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("StateMachine.Transitions() = %v, want %v", got, want)
	}
}

func TestStateMachine_EncodeDecode(t *testing.T) {
	sm := NewStateMachine("TestStateMachine_EncodeDecode", []Transition{
		{From: "foo", To: "bar", Handler: newRandomIDHandler(nil)},
		{From: "foo", To: "bax", Handler: newRandomIDHandler(nil)},
		{From: "bar", To: "baz", Handler: newRandomIDHandler(nil)},
	}, "foo")

	b, err := json.Marshal(sm)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	t.Logf("json.Marshal() = %s", b)

	t.Logf("handlers = %v", handlers)

	var got StateMachine
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !got.Equals(sm) {
		t.Errorf("json.Unmarshal() = %v, want %v", got, sm)
	}
}

func TestStateMachine_Clone(t *testing.T) {
	sm := NewStateMachine("TestStateMachine_Clone", []Transition{
		{From: "foo", To: "bar", Handler: newRandomIDHandler(nil)},
		{From: "foo", To: "bax", Handler: newRandomIDHandler(nil)},
		{From: "bar", To: "baz", Handler: newRandomIDHandler(nil)},
	}, "foo")

	got := sm.Clone()
	if !got.Equals(sm) {
		t.Errorf("StateMachine.Clone() = %v, want %v", got, sm)
	}
}

func ExampleStateMachine() {
	sm := NewStateMachine("ExampleStateMachine", []Transition{
		{From: "foo", To: "bar", Handler: newRandomIDHandler(func(from, to string, payload any) string {
			fmt.Println("from", from, "to", to, "payload", payload)
			return ""
		})},
		{From: "foo", To: "bax", Handler: newRandomIDHandler(func(from, to string, payload any) string {
			fmt.Println("from", from, "to", to, "payload", payload)
			return ""
		})},
		{From: "bar", To: "baz", Handler: newRandomIDHandler(func(from, to string, payload any) string {
			fmt.Println("from", from, "to", to, "payload", payload)
			return ""
		})},
	}, "foo")

	if err := sm.Transition("bar", nil); err != nil {
		panic(err)
	}

	if err := sm.Transition("baz", nil); err != nil {
		panic(err)
	}

	// Output:
	// from foo to bar payload <nil>
	// from bar to baz payload <nil>
}
