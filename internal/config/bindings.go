package config

var UserBindings = map[string]map[string]string{
	"buffer":   {},
	"command":  {},
	"terminal": {},
}

func SetUserBindings(parsed map[string]any) {
	UserBindings = map[string]map[string]string{
		"buffer":   {},
		"command":  {},
		"terminal": {},
	}

	for k, v := range parsed {
		switch val := v.(type) {
		case string:
			UserBindings["buffer"][k] = val
		case map[string]any:
			pane, ok := UserBindings[k]
			if !ok {
				continue
			}
			for key, action := range val {
				if str, ok := action.(string); ok {
					pane[key] = str
				}
			}
		}
	}
}
