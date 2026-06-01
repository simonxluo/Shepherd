package plugintest_test

import (
	"testing"

	"github.com/simonxluo/Shepherd/internal/backend"
	"github.com/simonxluo/Shepherd/internal/backend/plugintest"
)

func TestNewIsolatedRegistry_DoesNotTouchDefault(t *testing.T) {
	stub := &plugintest.FakePlugin{IDValue: "iso-only"}
	r := plugintest.NewIsolatedRegistry(stub)
	if _, ok := r.Get("iso-only"); !ok {
		t.Errorf("isolated registry missing the registered plugin")
	}
	if _, ok := backend.Default().Get("iso-only"); ok {
		t.Errorf("isolated registry leaked into Default()")
	}
}

func TestFakePlugin_Defaults(t *testing.T) {
	f := &plugintest.FakePlugin{}
	if f.ID() != "fake" {
		t.Errorf("default ID = %s, want fake", f.ID())
	}
	if f.DisplayName() != "Fake Plugin" {
		t.Errorf("default DisplayName = %q", f.DisplayName())
	}
	info, err := f.Discover(nil)
	if err != nil {
		t.Fatalf("Discover error: %v", err)
	}
	if !info.Available {
		t.Errorf("default Discover should return Available=true")
	}
	hr, err := f.CheckHealth(0)
	if err != nil {
		t.Fatalf("CheckHealth error: %v", err)
	}
	if !hr.Healthy {
		t.Errorf("default CheckHealth should be Healthy=true")
	}
	if !f.SupportsModel("anything") {
		t.Errorf("default SupportsModel should be true")
	}
	if vr := f.ValidateParams(nil); !vr.Valid {
		t.Errorf("default ValidateParams should be Valid=true")
	}
}

func TestFakePlugin_Overrides(t *testing.T) {
	f := &plugintest.FakePlugin{
		IDValue:                "custom",
		DiscoverFn:             func(*backend.Config) (*backend.Info, error) { return &backend.Info{Available: false}, nil },
		SupportsModelFn:        func(string) bool { return false },
		SupportedEndpointsMap:  map[string]bool{"/v1/chat/completions": true},
	}
	info, _ := f.Discover(nil)
	if info.Available {
		t.Errorf("override DiscoverFn ignored")
	}
	if f.SupportsModel("x") {
		t.Errorf("override SupportsModelFn ignored")
	}
	if got := f.SupportedEndpoints(); !got["/v1/chat/completions"] {
		t.Errorf("override SupportedEndpointsMap ignored: %v", got)
	}
}
