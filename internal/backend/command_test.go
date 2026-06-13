package backend

import (
	"strings"
	"testing"
)

func TestQuoteAndJoin_NoQuoting(t *testing.T) {
	got := quoteAndJoin([]string{"llama-server", "-m", "/models/foo.gguf", "-c", "4096"})
	want := "llama-server -m /models/foo.gguf -c 4096"
	if got != want {
		t.Errorf("quoteAndJoin = %q, want %q", got, want)
	}
}

func TestQuoteAndJoin_QuotesArgsWithSpaces(t *testing.T) {
	got := quoteAndJoin([]string{"vllm", "serve", "/models/dir with spaces/model"})
	if !strings.Contains(got, "'/models/dir with spaces/model'") {
		t.Errorf("expected single-quoted path, got %q", got)
	}
}

func TestQuoteAndJoin_EscapesSingleQuotes(t *testing.T) {
	got := quoteAndJoin([]string{"echo", "it's a test"})
	// "it's a test" -> 'it'\''s a test'
	if !strings.Contains(got, `'it'\''s a test'`) {
		t.Errorf("expected escaped single quote, got %q", got)
	}
}

func TestQuoteAndJoin_EmptyArgQuoted(t *testing.T) {
	got := quoteAndJoin([]string{"cmd", ""})
	if !strings.Contains(got, "''") {
		t.Errorf("expected empty arg to be quoted as '', got %q", got)
	}
}

func TestNewCommandSpec_PreviewIncludesQuoting(t *testing.T) {
	spec := NewCommandSpec("/usr/local/bin/vllm", []string{"serve", "/models/x y/model"}, nil, "")
	if !strings.Contains(spec.RedactedPreview, "'/models/x y/model'") {
		t.Errorf("preview missing quoted path: %q", spec.RedactedPreview)
	}
	if spec.Binary != "/usr/local/bin/vllm" {
		t.Errorf("Binary mismatch: %q", spec.Binary)
	}
	if spec.CommandLine() != spec.RedactedPreview {
		t.Errorf("CommandLine and RedactedPreview should match for non-secret commands")
	}
}

func TestCommandSpec_AppendRaw(t *testing.T) {
	spec := NewCommandSpec("vllm", []string{"serve", "/models/m"}, nil, "")
	out := spec.AppendRaw("--enforce-eager --disable-log-requests")
	if len(out.Args) != 4 {
		t.Fatalf("expected 4 args after AppendRaw, got %d (%v)", len(out.Args), out.Args)
	}
	if out.Args[2] != "--enforce-eager" || out.Args[3] != "--disable-log-requests" {
		t.Errorf("AppendRaw args wrong: %v", out.Args)
	}
	if !strings.Contains(out.RedactedPreview, "--enforce-eager") {
		t.Errorf("preview missing appended arg: %q", out.RedactedPreview)
	}
}

func TestCommandSpec_AppendRaw_EmptyNoOp(t *testing.T) {
	spec := NewCommandSpec("vllm", []string{"serve"}, nil, "")
	out := spec.AppendRaw("   ")
	if len(out.Args) != 1 {
		t.Errorf("AppendRaw with whitespace-only string mutated args: %v", out.Args)
	}
}
