// Package sm provides a simple finite state machine implementation.
package sm

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

var (
	handlers      = make(map[string]map[string]Handler)
	handlersMutex sync.RWMutex
)

func register(name, id string, handler Handler) {
	handlersMutex.Lock()
	defer handlersMutex.Unlock()

	if handlers[name] == nil {
		handlers[name] = make(map[string]Handler)
	}
	handlers[name][id] = handler
}

func handler(name, id string) Handler {
	handlersMutex.RLock()
	defer handlersMutex.RUnlock()

	return handlers[name][id]
}

type illegalTransitionError struct {
	current, next string
}

func (e illegalTransitionError) Error() string {
	current := e.current
	if current == "" {
		current = "<nil>"
	}
	return fmt.Sprintf("illegal transition: <%s> -x-> <%s>", current, e.next)
}

// HandleTransition is a function that is called when the state machine
type HandleTransition func(from, to string, payload any) string

// Handler is an interface that can be implemented to handle state transitions.
type Handler struct {
	// ID returns the unique identifier of the handler.
	ID string

	// H is called when the state machine transitions from one
	// state to another.
	H HandleTransition
}

// NewHandler returns a new Handler.
func NewHandler(id string, h HandleTransition) Handler {
	return Handler{ID: id, H: h}
}

// Transition is a transition from one state to another.
type Transition struct {
	From    string
	To      string
	Handler Handler
}

// NewTransition returns a new Transition.
func NewTransition(from, to string, handlers ...Handler) Transition {
	var handler Handler
	if len(handlers) > 0 {
		handler = handlers[0]
	}
	return Transition{
		From:    from,
		To:      to,
		Handler: handler,
	}
}

// StateMachine is a simple state machine implementation.
// It is designed to be used in a concurrent environment, and is safe for
// concurrent use.
// The zero value of StateMachine is a valid state machine.
type StateMachine struct {
	name            string
	current         string
	transitions     map[string]map[string]Handler
	terminateStates []string
}

// NewStateMachine returns a new StateMachine.
// The initial state is set to the first state in the states slice.
func NewStateMachine(name string, transitions []Transition, current string) *StateMachine {
	sm := &StateMachine{
		name:        name,
		transitions: make(map[string]map[string]Handler),
	}

	tos := map[string]struct{}{}

	for _, t := range transitions {
		if sm.transitions[t.From] == nil {
			sm.transitions[t.From] = make(map[string]Handler)
		}
		sm.transitions[t.From][t.To] = t.Handler

		if t.Handler.ID != "" {
			register(name, t.Handler.ID, t.Handler)
		}

		tos[t.To] = struct{}{}
	}

	for to := range tos {
		if _, ok := sm.transitions[to]; !ok {
			sm.terminateStates = append(sm.terminateStates, to)
		}
	}

	if err := sm.SetCurrent(current); err != nil {
		panic(err)
	}

	return sm
}

// TerminateStates returns the list of states that the state machine can
// terminate in.
func (sm *StateMachine) TerminateStates() []string {
	return sm.terminateStates
}

// IsTerminated returns true if the state machine is in a terminate state.
func (sm *StateMachine) IsTerminated() bool {
	for _, state := range sm.terminateStates {
		if state == sm.current {
			return true
		}
	}
	return false
}

// Name returns the name of the state machine.
func (sm *StateMachine) Name() string {
	return sm.name
}

// Current returns the current state of the state machine.
func (sm *StateMachine) Current() string {
	return sm.current
}

// SetCurrent sets the current state of the state machine.
func (sm *StateMachine) SetCurrent(state string) error {
	if _, ok := sm.transitions[state]; !ok {
		return illegalTransitionError{sm.current, state}
	}

	sm.current = state
	return nil
}

// Transition transitions the state machine from the current state to the next
// state.
// If the transition is not allowed, an error is returned.
func (sm *StateMachine) Transition(next string, payload any) error {
	handler, ok := sm.transitions[sm.current][next]
	if !ok {
		return illegalTransitionError{sm.current, next}
	}

	if handler.H != nil {
		if unexpected := handler.H(sm.current, next, payload); unexpected != "" {
			next = unexpected
		}
	}

	sm.current = next
	return nil
}

// CanTransition returns true if the state machine can transition from the
// current state to the next state.
func (sm *StateMachine) CanTransition(next string) bool {
	_, ok := sm.transitions[sm.current][next]
	return ok
}

// Transitions returns a list of states that the state machine can transition to
// from the current state.
// The returned list is a copy of the internal list, and modifying it will not
// affect the state machine.
func (sm *StateMachine) Transitions() []string {
	transitions := make([]string, 0, len(sm.transitions[sm.current]))
	for state := range sm.transitions[sm.current] {
		transitions = append(transitions, state)
	}
	sort.Strings(transitions)
	return transitions
}

// Equals returns true if the state machine has the same current state as the
// other state machine. and the same transitions.
func (sm *StateMachine) Equals(other *StateMachine) bool {
	if sm == nil && other == nil {
		return true
	}

	if sm == nil || other == nil {
		return false
	}

	if sm.name != other.name {
		return false
	}

	if sm.current != other.current {
		return false
	}

	if len(sm.transitions) != len(other.transitions) {
		return false
	}

	for from, tos := range sm.transitions {
		otherTos, ok := other.transitions[from]
		if !ok {
			return false
		}

		if len(tos) != len(otherTos) {
			return false
		}

		for to, toH := range tos {
			otherToH, ok := otherTos[to]
			if !ok {
				return false
			}

			if toH.ID != otherToH.ID {
				return false
			}
		}
	}

	return true
}

// Clone returns a new StateMachine that is a copy of the current state machine.
func (sm *StateMachine) Clone() *StateMachine {
	clone := &StateMachine{
		name:        sm.name,
		current:     sm.current,
		transitions: make(map[string]map[string]Handler),
	}

	for from, tos := range sm.transitions {
		clone.transitions[from] = make(map[string]Handler)
		for to, handler := range tos {
			clone.transitions[from][to] = handler
		}
	}

	return clone
}

type snapshot struct {
	Name        string
	Current     string
	Transitions map[string]map[string]string
}

func (sm *StateMachine) MarshalJSON() ([]byte, error) {
	var transitions = map[string]map[string]string{}

	for from, tos := range sm.transitions {
		transitions[from] = map[string]string{}
		for to, toH := range tos {
			transitions[from][to] = toH.ID
		}
	}

	return json.Marshal(snapshot{Name: sm.name, Current: sm.current, Transitions: transitions})
}

var _ json.Marshaler = (*StateMachine)(nil)

func (sm *StateMachine) UnmarshalJSON(data []byte) error {
	var s snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	handlersMutex.RLock()
	defer handlersMutex.RUnlock()

	sm.name = s.Name
	sm.current = s.Current
	sm.transitions = map[string]map[string]Handler{}

	for from, tos := range s.Transitions {
		sm.transitions[from] = map[string]Handler{}
		for to, id := range tos {
			sm.transitions[from][to] = handler(sm.name, id)
		}
	}

	return nil
}

var _ json.Unmarshaler = (*StateMachine)(nil)
