package kernel

import (
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if nn := strings.Count(Version(), "."); nn != 2 {
		t.Errorf("invalid version count %v", nn)
	}
}
