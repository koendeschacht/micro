package action

import "time"

type KeySequenceTimeoutEvent struct {
	when   time.Time
	paneID uint64
	token  uint64
}

type TextObjectPreviewEvent struct {
	when   time.Time
	paneID uint64
	token  uint64
}

func NewKeySequenceTimeoutEvent(paneID uint64, token uint64) *KeySequenceTimeoutEvent {
	return &KeySequenceTimeoutEvent{
		when:   time.Now(),
		paneID: paneID,
		token:  token,
	}
}

func (e *KeySequenceTimeoutEvent) When() time.Time {
	return e.when
}

func (e *KeySequenceTimeoutEvent) EscSeq() string {
	return ""
}

func NewTextObjectPreviewEvent(paneID uint64, token uint64) *TextObjectPreviewEvent {
	return &TextObjectPreviewEvent{
		when:   time.Now(),
		paneID: paneID,
		token:  token,
	}
}

func (e *TextObjectPreviewEvent) When() time.Time {
	return e.when
}

func (e *TextObjectPreviewEvent) EscSeq() string {
	return ""
}
