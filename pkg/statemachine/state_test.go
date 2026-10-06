package statemachine

import (
	"testing"
)

func TestStateMachine(t *testing.T) {
	sm := NewStateMachine()

	if sm.Current() != StateIdle {
		t.Fatalf("expected initial state IDLE, got %s", sm.Current())
	}

	// Wake word trigger
	if !sm.WakeWordTriggered("hey voxlab", 0.95) {
		t.Fatalf("expected transition to COMMAND_LISTEN")
	}
	if sm.Current() != StateCommandListen {
		t.Errorf("expected state COMMAND_LISTEN, got %s", sm.Current())
	}

	// Endpointing
	if !sm.SpeechEndpointed("open settings") {
		t.Fatalf("expected transition to PROCESS")
	}
	if sm.Current() != StateProcess {
		t.Errorf("expected state PROCESS, got %s", sm.Current())
	}

	// Completion
	sm.FinishProcess()
	if sm.Current() != StateIdle {
		t.Errorf("expected state IDLE, got %s", sm.Current())
	}

	// Dictation flow
	if !sm.StartDictation("camera_1") {
		t.Fatalf("expected transition to ANNOTATION_LISTEN")
	}
	if sm.Current() != StateAnnotationListen {
		t.Errorf("expected state ANNOTATION_LISTEN, got %s", sm.Current())
	}

	sm.SpeechEndpointed("inspection completed")
	if sm.Current() != StateReview {
		t.Errorf("expected state REVIEW, got %s", sm.Current())
	}

	sm.FinishReview(true)
	if sm.Current() != StateIdle {
		t.Errorf("expected state IDLE, got %s", sm.Current())
	}

	// Cancel
	sm.StartCommand()
	sm.Cancel()
	if sm.Current() != StateIdle {
		t.Errorf("expected state IDLE after cancel, got %s", sm.Current())
	}
}
