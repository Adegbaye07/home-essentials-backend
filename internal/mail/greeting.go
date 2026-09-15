package mail

import "strings"

func greetingLine(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Hi,"
	}
	return "Hi " + name + ","
}
