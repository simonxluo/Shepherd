package backend

import (
	"os"
	"strings"
	"testing"
)

func TestBuildEnvWithVars_NilReturnsOSEnv(t *testing.T) {
	env := BuildEnvWithVars(nil)
	osEnv := os.Environ()
	if len(env) != len(osEnv) {
		t.Errorf("BuildEnvWithVars(nil) length = %d, os.Environ() = %d", len(env), len(osEnv))
	}
}

func TestBuildEnvWithVars_AddsAndOverrides(t *testing.T) {
	t.Setenv("BACKEND_TEST_EXISTING", "old")
	env := BuildEnvWithVars([]string{
		"BACKEND_TEST_NEW=hello",
		"BACKEND_TEST_EXISTING=new",
	})
	var sawNew, sawNewVal, sawExistingNew bool
	for _, kv := range env {
		switch {
		case kv == "BACKEND_TEST_NEW=hello":
			sawNew = true
			sawNewVal = true
		case kv == "BACKEND_TEST_EXISTING=new":
			sawExistingNew = true
		case strings.HasPrefix(kv, "BACKEND_TEST_EXISTING=old"):
			t.Errorf("override did not replace old value: %q", kv)
		}
	}
	if !sawNew || !sawNewVal {
		t.Errorf("new variable not appended")
	}
	if !sawExistingNew {
		t.Errorf("existing variable not overridden")
	}
}

func TestBuildEnvWithVars_SkipsMalformed(t *testing.T) {
	env := BuildEnvWithVars([]string{
		"",
		"NO_EQUALS_SIGN",
		"=NO_NAME",
		"GOOD=ok",
	})
	var sawGood bool
	for _, kv := range env {
		if kv == "GOOD=ok" {
			sawGood = true
		}
		if kv == "NO_EQUALS_SIGN" || strings.HasPrefix(kv, "=NO_NAME") {
			t.Errorf("malformed entry leaked into env: %q", kv)
		}
	}
	if !sawGood {
		t.Errorf("good entry was dropped")
	}
}
