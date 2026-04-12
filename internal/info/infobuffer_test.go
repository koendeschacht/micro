package info

import (
	"testing"
	"time"

	"github.com/micro-editor/micro/v2/internal/config"
)

func withGlobalSettings(settings map[string]any, fn func()) {
	prev := config.GlobalSettings
	config.GlobalSettings = settings
	defer func() {
		config.GlobalSettings = prev
	}()
	fn()
}

func TestTransientMessageDurationDefaults(t *testing.T) {
	withGlobalSettings(map[string]any{}, func() {
		if got := transientMessageDuration(MsgInfo); got != defaultInfoMessageDuration {
			t.Fatalf("info message duration = %v, want %v", got, defaultInfoMessageDuration)
		}
		if got := transientMessageDuration(MsgSuccess); got != defaultInfoMessageDuration {
			t.Fatalf("success message duration = %v, want %v", got, defaultInfoMessageDuration)
		}
		if got := transientMessageDuration(MsgError); got != defaultErrorMessageDuration {
			t.Fatalf("error message duration = %v, want %v", got, defaultErrorMessageDuration)
		}
	})
}

func TestTransientMessageDurationConfigured(t *testing.T) {
	withGlobalSettings(map[string]any{
		"infomessagetimeout":  float64(1.5),
		"errormessagetimeout": float64(7),
	}, func() {
		if got := transientMessageDuration(MsgInfo); got != 1500*time.Millisecond {
			t.Fatalf("info message duration = %v, want %v", got, 1500*time.Millisecond)
		}
		if got := transientMessageDuration(MsgError); got != 7*time.Second {
			t.Fatalf("error message duration = %v, want %v", got, 7*time.Second)
		}
	})
}

func TestZeroMessageTimeoutDoesNotExpire(t *testing.T) {
	withGlobalSettings(map[string]any{
		"infomessagetimeout":  float64(0),
		"errormessagetimeout": float64(0),
	}, func() {
		ib := &InfoBuf{}
		ib.Message("hello")

		if !ib.ExpiresAt.IsZero() {
			t.Fatalf("expected persistent info message, got expiry at %v", ib.ExpiresAt)
		}
		ib.ExpireMessage()
		if ib.Msg != "hello" || !ib.HasMessage {
			t.Fatalf("persistent info message expired unexpectedly: msg=%q hasMessage=%v", ib.Msg, ib.HasMessage)
		}

		ib.Error("boom")
		if !ib.ExpiresAt.IsZero() {
			t.Fatalf("expected persistent error message, got expiry at %v", ib.ExpiresAt)
		}
		ib.ExpireMessage()
		if ib.Msg != "boom" || !ib.HasError {
			t.Fatalf("persistent error message expired unexpectedly: msg=%q hasError=%v", ib.Msg, ib.HasError)
		}
	})
}
