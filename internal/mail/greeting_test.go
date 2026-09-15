package mail

import "testing"

func TestGreetingLine(t *testing.T) {
	if got := greetingLine("Ada"); got != "Hi Ada," {
		t.Fatalf("got %q", got)
	}
	if got := greetingLine("  "); got != "Hi," {
		t.Fatalf("empty name: %q", got)
	}
}
