package dumper

var (
	defaultStyles = map[string]string{}
	colorStyles   = map[string]string{
		"default":   "38;5;208",
		"num":       "1;38;5;38",
		"const":     "1;38;5;208",
		"str":       "1;38;5;113",
		"note":      "38;5;38",
		"ref":       "38;5;245",
		"public":    "",
		"protected": "",
		"private":   "",
		"meta":      "38;5;170",
		"key":       "38;5;113",
		"index":     "38;5;38",
	}
)
