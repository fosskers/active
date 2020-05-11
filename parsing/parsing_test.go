package parsing

import "testing"

func TestParseAction(t *testing.T) {
	action := parseAction("uses: actions/checkout@v2")
	expected := Action{"actions", "checkout", "2"}
	if action != expected {
		t.Errorf("parseAction: expected %s, got %s", expected, action)
	}
}

func TestRaw(t *testing.T) {
	raw := Action{"actions", "checkout", "2"}.Raw()
	expected := "actions/checkout@v2"
	if raw != expected {
		t.Errorf("Raw: expected %s, got %s", expected, raw)
	}
}

func TestActions(t *testing.T) {
	actions := Actions("uses: actions/checkout@v2\n   uses: actions/cache@v1")
	expected := []Action{
		{"actions", "checkout", "2"},
		{"actions", "cache", "1"}}
	for i, v := range actions {
		if v != expected[i] {
			t.Errorf("Raw: expected %s, got %s", expected[i], v)
		}
	}
}
