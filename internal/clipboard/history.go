package clipboard

const maxHistoryEntries = 10

var history []string

func recordHistory(text string) {
	history = append([]string{text}, history...)
	if len(history) > maxHistoryEntries {
		history = history[:maxHistoryEntries]
	}
}

// GetHistory returns the clipboard write history, newest first.
func GetHistory() []string {
	if len(history) == 0 {
		return nil
	}
	out := make([]string, len(history))
	copy(out, history)
	return out
}
