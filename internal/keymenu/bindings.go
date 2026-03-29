package keymenu

type Token struct {
	Key   string
	Ctrl  bool
	Alt   bool
	Shift bool
	Meta  bool
	Mouse bool
}

type Binding struct {
	Pane      string
	Action    string
	Canonical string
	Sequence  []Token
}

var resolvedBindings = map[string]map[string]Binding{
	"buffer":   {},
	"command":  {},
	"terminal": {},
}

func ResetBindings() {
	resolvedBindings = map[string]map[string]Binding{
		"buffer":   {},
		"command":  {},
		"terminal": {},
	}
}

func SetBinding(pane, canonical, action string, sequence []Token) {
	paneBindings, ok := resolvedBindings[pane]
	if !ok {
		return
	}
	paneBindings[canonical] = Binding{
		Pane:      pane,
		Action:    action,
		Canonical: canonical,
		Sequence:  append([]Token(nil), sequence...),
	}
}

func PaneBindings(pane string) []Binding {
	paneBindings, ok := resolvedBindings[pane]
	if !ok {
		return nil
	}
	out := make([]Binding, 0, len(paneBindings))
	for _, binding := range paneBindings {
		out = append(out, binding)
	}
	return out
}

func PreferredBindingForAction(pane, action string) *Binding {
	bindings := PaneBindings(pane)
	var best *Binding
	for i := range bindings {
		binding := &bindings[i]
		if !BindingIncludesAction(binding.Action, action) {
			continue
		}
		if best == nil || len(binding.Sequence) < len(best.Sequence) || (len(binding.Sequence) == len(best.Sequence) && binding.Canonical < best.Canonical) {
			best = binding
		}
	}
	return best
}

func BindingIncludesAction(bindingAction, target string) bool {
	for _, token := range bindingTokens(bindingAction) {
		if token == target {
			return true
		}
	}
	return false
}

func bindingTokens(action string) []string {
	var out []string
	start := 0
	for i, r := range action {
		if r == '&' || r == '|' || r == ',' {
			if start < i {
				out = append(out, trimToken(action[start:i]))
			}
			start = i + 1
		}
	}
	if start < len(action) {
		out = append(out, trimToken(action[start:]))
	}
	return out
}

func trimToken(token string) string {
	start := 0
	for start < len(token) && token[start] == ' ' {
		start++
	}
	end := len(token)
	for end > start && token[end-1] == ' ' {
		end--
	}
	return token[start:end]
}

func EqualTokens(left, right Token) bool {
	return left.Key == right.Key && left.Ctrl == right.Ctrl && left.Alt == right.Alt && left.Shift == right.Shift && left.Meta == right.Meta && left.Mouse == right.Mouse
}

func HasPrefix(sequence, prefix []Token) bool {
	if len(sequence) < len(prefix) {
		return false
	}
	for i := range prefix {
		if !EqualTokens(sequence[i], prefix[i]) {
			return false
		}
	}
	return true
}
