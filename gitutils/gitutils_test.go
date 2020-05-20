package gitutils

import "testing"

func TestVStrip(t *testing.T) {
	version := versionFormat("v1.2.3")
	if version != "1.2.3" {
		t.Errorf("versionFormat(v1.2.3) ?= 1.2.3, got %s", version)
	}
}
