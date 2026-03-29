package action

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v3"
	"github.com/micro-editor/micro/v2/internal/keymenu"
)

func keyMenuTokenFromKeyEvent(k KeyEvent) keymenu.Token {
	name := ""
	if keyName, ok := tcell.KeyNames[k.code]; ok {
		name = keyName
	} else if k.code == tcell.KeyRune {
		name = string(k.r)
	} else {
		name = fmt.Sprintf("Key[%d]", k.code)
	}

	token := keymenu.Token{
		Key:   name,
		Ctrl:  k.mod&tcell.ModCtrl != 0,
		Alt:   k.mod&tcell.ModAlt != 0,
		Shift: k.mod&tcell.ModShift != 0,
		Meta:  k.mod&tcell.ModMeta != 0,
	}

	for {
		switched := false
		switch {
		case strings.HasPrefix(token.Key, "Ctrl-"):
			token.Ctrl = true
			token.Key = strings.TrimPrefix(token.Key, "Ctrl-")
			switched = true
		case strings.HasPrefix(token.Key, "Alt-"):
			token.Alt = true
			token.Key = strings.TrimPrefix(token.Key, "Alt-")
			switched = true
		case strings.HasPrefix(token.Key, "Shift-"):
			token.Shift = true
			token.Key = strings.TrimPrefix(token.Key, "Shift-")
			switched = true
		case strings.HasPrefix(token.Key, "Meta-"):
			token.Meta = true
			token.Key = strings.TrimPrefix(token.Key, "Meta-")
			switched = true
		}
		if !switched {
			break
		}
	}

	if k.code == tcell.KeyRune && len(token.Key) == 1 {
		token.Key = string(unicode.ToLower(rune(token.Key[0])))
	}

	return token
}

func keyMenuTokenFromEvent(event Event) keymenu.Token {
	switch ev := event.(type) {
	case KeyEvent:
		return keyMenuTokenFromKeyEvent(ev)
	case MouseEvent:
		return keymenu.Token{Key: ev.Name(), Mouse: true}
	default:
		return keymenu.Token{Key: event.Name()}
	}
}

func keyMenuSequenceFromEvent(event Event) []keymenu.Token {
	switch ev := event.(type) {
	case KeySequenceEvent:
		sequence := make([]keymenu.Token, 0, len(ev.keys))
		for _, part := range ev.keys {
			sequence = append(sequence, keyMenuTokenFromEvent(part))
		}
		return sequence
	default:
		return []keymenu.Token{keyMenuTokenFromEvent(event)}
	}
}
