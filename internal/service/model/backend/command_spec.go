package backend

import "strings"

// CommandSpec is the structured representation of a backend launch command.
type CommandSpec struct {
	Binary          string   `json:"binary"`
	Args            []string `json:"args"`
	Env             []string `json:"env,omitempty"`
	WorkDir         string   `json:"workDir,omitempty"`
	RedactedPreview string   `json:"redactedPreview"`
}

// NewCommandSpec builds a command spec and its shell-preview string.
func NewCommandSpec(binary string, args []string, env []string, workDir string) CommandSpec {
	argv := make([]string, 0, 1+len(args))
	argv = append(argv, binary)
	argv = append(argv, args...)
	return CommandSpec{
		Binary:          binary,
		Args:            append([]string(nil), args...),
		Env:             append([]string(nil), env...),
		WorkDir:         workDir,
		RedactedPreview: quoteAndJoin(argv),
	}
}

// CommandLine serializes the spec to the current process manager command format.
func (s CommandSpec) CommandLine() string {
	parts := make([]string, 0, 1+len(s.Args))
	parts = append(parts, s.Binary)
	parts = append(parts, s.Args...)
	return quoteAndJoin(parts)
}

// AppendRaw appends raw extra CLI text to the preview/command line.
func (s CommandSpec) AppendRaw(raw string) CommandSpec {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return s
	}
	if s.RedactedPreview == "" {
		s.RedactedPreview = raw
	} else {
		s.RedactedPreview += " " + raw
	}
	return s
}
