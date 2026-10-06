package statemachine

import (
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
	mu           sync.RWMutex
	current      State
	listeners    []func(event StateEvent)
	turnStart    time.Time
	activeTarget string
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

func (sm *StateMachine) transition(to State, trigger string, data interface{}) bool {
	sm.mu.Lock()
	from := sm.current
	if from == to {
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
	sm.mu.RLock()
	cur := sm.current
	sm.mu.RUnlock()
	if cur == StateIdle || cur == StateFault {
		return sm.transition(StateCommandListen, "WAKE_WORD_DETECTED", map[string]interface{}{
			"phrase":     phrase,
			"confidence": confidence,
		})
	}
	return false
}

// StartCommand transitions from IDLE to COMMAND_LISTEN (via push-to-talk button).
func (sm *StateMachine) StartCommand() bool {
	sm.mu.RLock()
	cur := sm.current
	sm.mu.RUnlock()
	if cur == StateIdle || cur == StateFault {
		return sm.transition(StateCommandListen, "MANUAL_ACTIVATION", nil)
	}
	return false
}

// StartDictation transitions to ANNOTATION_LISTEN mode.
func (sm *StateMachine) StartDictation(targetID string) bool {
	sm.mu.Lock()
	sm.activeTarget = targetID
	sm.mu.Unlock()
	return sm.transition(StateAnnotationListen, "START_DICTATION", targetID)
}

// SpeechEndpointed transitions from LISTEN -> PROCESS when user finishes talking.
func (sm *StateMachine) SpeechEndpointed(transcript string) bool {
	sm.mu.RLock()
	cur := sm.current
	sm.mu.RUnlock()
	if cur == StateCommandListen {
		return sm.transition(StateProcess, "SPEECH_ENDPOINT", transcript)
	} else if cur == StateAnnotationListen {
		return sm.transition(StateReview, "DICTATION_ENDPOINT", transcript)
	}
	return false
}

// RequestConfirmation moves from PROCESS to CONFIRM.
func (sm *StateMachine) RequestConfirmation(intentID string) bool {
	return sm.transition(StateConfirm, "REQUIRE_CONFIRMATION", intentID)
}

// ConfirmAction completes confirmation and returns to IDLE.
func (sm *StateMachine) ConfirmAction(approved bool) bool {
	trigger := "ACTION_DECLINED"
	if approved {
		trigger = "ACTION_CONFIRMED"
	}
	return sm.transition(StateIdle, trigger, nil)
}

// FinishProcess returns to IDLE after action executed or rejected.
func (sm *StateMachine) FinishProcess() bool {
	return sm.transition(StateIdle, "PROCESS_COMPLETED", nil)
}

// FinishReview returns to IDLE after draft saved or discarded.
func (sm *StateMachine) FinishReview(saved bool) bool {
	trigger := "DRAFT_DISCARDED"
	if saved {
		trigger = "DRAFT_SAVED"
	}
	return sm.transition(StateIdle, trigger, nil)
}

// SetSpeaking transitions to/from SPEAKING state for half-duplex TTS.
func (sm *StateMachine) SetSpeaking(speaking bool) bool {
	if speaking {
		return sm.transition(StateSpeaking, "TTS_START", nil)
	}
	return sm.transition(StateIdle, "TTS_FINISH", nil)
}

// Cancel returns system immediately to IDLE from any active state.
func (sm *StateMachine) Cancel() bool {
	return sm.transition(StateIdle, "CANCEL_REQUESTED", nil)
}

// Fault transitions system to FAULT.
func (sm *StateMachine) Fault(errMsg string) bool {
	return sm.transition(StateFault, "ERROR_OCCURRED", errMsg)
}
