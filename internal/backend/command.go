package backend

import "strings"

// CommandSpec is the structured form of a shell command line: the binary, the
// list of already-tokenised arguments, environment variables, optional
// working directory, and a redacted preview suitable for logging.
//
// The supervisor consumes Binary + Args directly (no shell interpolation);
// CommandLine() and RedactedPreview are for logs and the /api/.../preview
// endpoint.
type CommandSpec struct {
	Binary          string
	Args            []string
	Env             []string
	WorkDir         string
	RedactedPreview string
}

// NewCommandSpec creates a CommandSpec, populating RedactedPreview from a
// shell-quoted join of binary + args.
func NewCommandSpec(binary string, args []string, env []string, workDir string) CommandSpec {
	full := append([]string{binary}, args...)
	return CommandSpec{
		Binary:          binary,
		Args:            args,
		Env:             env,
		WorkDir:         workDir,
		RedactedPreview: quoteAndJoin(full),
	}
}

// CommandLine returns the spec rendered as a single shell-quoted string.
func (s CommandSpec) CommandLine() string {
	full := append([]string{s.Binary}, s.Args...)
	return quoteAndJoin(full)
}

// AppendRaw appends extra raw arguments (already shell-quoted by the caller)
// to the command preview. Used for global / model-level extra_args passthrough.
//
// The returned CommandSpec has the new tail in Args (split naively on
// whitespace) and an updated RedactedPreview.
func (s CommandSpec) AppendRaw(raw string) CommandSpec {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return s
	}
	// Split on whitespace; quoted args inside extra_args remain a TODO if
	// users ever pass spaces inside a single arg. Callers must escape such
	// args themselves.
	extra := strings.Fields(raw)
	out := s
	out.Args = append(append([]string{}, s.Args...), extra...)
	full := append([]string{out.Binary}, out.Args...)
	out.RedactedPreview = quoteAndJoin(full)
	return out
}

// quoteAndJoin shell-quotes each argument as needed and joins with spaces.
// Used by CommandSpec, plugins' command builders, and the preview endpoint.
func quoteAndJoin(args []string) string {
	parts := make([]string, 0, len(args))
	for _, a := range args {
		if needsQuoting(a) {
			parts = append(parts, "'"+escapeQuotes(a)+"'")
		} else {
			parts = append(parts, a)
		}
	}
	return strings.Join(parts, " ")
}

// QuoteAndJoin is the exported helper that plugin subpackages call; the
// unexported quoteAndJoin is used by CommandSpec methods inside this package.
func QuoteAndJoin(args []string) string { return quoteAndJoin(args) }

func needsQuoting(arg string) bool {
	if arg == "" {
		return true
	}
	for _, r := range arg {
		switch r {
		case ' ', '\t', '\n', '"', '\'', '\\', '$', '`', '|', '&', ';', '<', '>', '(', ')', '[', ']', '{', '}', '*', '?', '#', '~', '!':
			return true
		}
	}
	return false
}

func escapeQuotes(s string) string {
	// Single-quote-wrapped strings: replace internal ' with '\''
	return strings.ReplaceAll(s, "'", `'\''`)
}
