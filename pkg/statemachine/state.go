package statemachine

import (
	"slices"
	"sync"
	"time"
)

// State represents the voice interaction state.
type State string

const (
	StateIdle             State = "IDLE"
	StateCommandListen    State = "COMMAND_LISTEN"
	StateProcess          State = "PROCESS"
	StateConfirm          State = "CONFIRM"
	StateAnnotationListen State = "ANNOTATION_LISTEN"
	StateReview           State = "REVIEW"
	StateSpeaking         State = "SPEAKING"
	StateFault            State = "FAULT"
)

// StateEvent represents an event triggering a state transition.
type StateEvent struct {
	FromState State       `json:"from_state"`
	ToState   State       `json:"to_state"`
	Trigger   string      `json:"trigger"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// StateMachine manages the lifecycle of voice interactions.
type StateMachine struct {
	mu        sync.RWMutex
	current   State
	listeners []func(event StateEvent)
	turnStart time.Time
}

// NewStateMachine creates an initialized state machine.
func NewStateMachine() *StateMachine {
	return &StateMachine{
		current: StateIdle,
	}
}

// Current returns the current state.
func (sm *StateMachine) Current() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.current
}

// AddListener registers a callback invoked on any state transition.
func (sm *StateMachine) AddListener(fn func(event StateEvent)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.listeners = append(sm.listeners, fn)
}

// transition moves to `to` if the current state is one of allowedFrom (any state when
// allowedFrom is empty). The source check and the assignment share one critical section,
// so concurrent callers cannot invalidate a guard between check and act.
func (sm *StateMachine) transition(to State, trigger string, data interface{}, allowedFrom ...State) bool {
	sm.mu.Lock()
	from := sm.current
	if from == to || (len(allowedFrom) > 0 && !slices.Contains(allowedFrom, from)) {
		sm.mu.Unlock()
		return false
	}
	sm.current = to
	sm.turnStart = time.Now()
	event := StateEvent{
		FromState: from,
		ToState:   to,
		Trigger:   trigger,
		Timestamp: sm.turnStart,
		Data:      data,
	}
	listeners := append([]func(event StateEvent){}, sm.listeners...)
	sm.mu.Unlock()

	for _, l := range listeners {
		l(event)
	}
	return true
}

// WakeWordTriggered transitions IDLE -> COMMAND_LISTEN upon wake phrase detection.
func (sm *StateMachine) WakeWordTriggered(phrase string, confidence float64) bool {
	return sm.transition(StateCommandListen, "WAKE_WORD_DETECTED", map[string]interface{}{
		"phrase":     phrase,
		"confidence": confidence,
	}, StateIdle, StateFault)
}

// StartCommand transitions from IDLE to COMMAND_LISTEN (via push-to-talk button).
func (sm *StateMachine) StartCommand() bool {
	return sm.transition(StateCommandListen, "MANUAL_ACTIVATION", nil, StateIdle, StateFault)
}

// StartDictation transitions to ANNOTATION_LISTEN mode. Like Cancel it is an explicit
// user action, so it is allowed from any state.
func (sm *StateMachine) StartDictation(targetID string) bool {
	return sm.transition(StateAnnotationListen, "START_DICTATION", targetID)
}

// SpeechEndpointed transitions from LISTEN -> PROCESS when user finishes talking.
func (sm *StateMachine) SpeechEndpointed(transcript string) bool {
	return sm.transition(StateProcess, "SPEECH_ENDPOINT", transcript, StateCommandListen) ||
		sm.transition(StateReview, "DICTATION_ENDPOINT", transcript, StateAnnotationListen)
}

// RequestConfirmation moves from PROCESS to CONFIRM. IDLE is allowed too: typed
// (injected) commands are matched without a listening turn.
func (sm *StateMachine) RequestConfirmation(intentID string) bool {
	return sm.transition(StateConfirm, "REQUIRE_CONFIRMATION", intentID, StateProcess, StateIdle)
}

// ConfirmAction completes confirmation and returns to IDLE.
func (sm *StateMachine) ConfirmAction(approved bool) bool {
	trigger := "ACTION_DECLINED"
	if approved {
		trigger = "ACTION_CONFIRMED"
	}
	return sm.transition(StateIdle, trigger, nil, StateConfirm)
}

// FinishProcess returns to IDLE after action executed or rejected.
func (sm *StateMachine) FinishProcess() bool {
	return sm.transition(StateIdle, "PROCESS_COMPLETED", nil, StateProcess)
}

// FinishReview returns to IDLE after draft saved or discarded. Stopping dictation
// finalizes straight from ANNOTATION_LISTEN, so that is a valid source too.
func (sm *StateMachine) FinishReview(saved bool) bool {
	trigger := "DRAFT_DISCARDED"
	if saved {
		trigger = "DRAFT_SAVED"
	}
	return sm.transition(StateIdle, trigger, nil, StateReview, StateAnnotationListen)
}

// SetSpeaking transitions to/from SPEAKING state for half-duplex TTS.
func (sm *StateMachine) SetSpeaking(speaking bool) bool {
	if speaking {
		// Playback during a listening turn must not end it; echo muting applies regardless.
		return sm.transition(StateSpeaking, "TTS_START", nil, StateIdle)
	}
	// Playback-stopped reports also arrive when nothing was playing (e.g. UI resets);
	// they must not abort a listening session.
	return sm.transition(StateIdle, "TTS_FINISH", nil, StateSpeaking)
}

// Cancel returns system immediately to IDLE from any active state.
func (sm *StateMachine) Cancel() bool {
	return sm.transition(StateIdle, "CANCEL_REQUESTED", nil)
}

// Fault transitions system to FAULT.
func (sm *StateMachine) Fault(errMsg string) bool {
	return sm.transition(StateFault, "ERROR_OCCURRED", errMsg)
}
