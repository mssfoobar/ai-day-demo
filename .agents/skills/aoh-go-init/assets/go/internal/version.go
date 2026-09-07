package internal

var Version string

func init() {
	if Version == "" {
		Version = "Dev"
	}
}
