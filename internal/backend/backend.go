// Package backend defines the plugin contract for inference backends and provides
// the registry, resolver, and shared utilities used by all built-in plugins.
//
// A "plugin" is a self-contained implementation of the Plugin interface that
// knows how to:
//   - discover whether the underlying engine binary is installed (Discover);
//   - turn a model load request into a runnable command (BuildStartConfig);
//   - decide when the spawned process is ready to serve (IsLoadComplete);
//   - probe its health (CheckHealth);
//   - report which API endpoints it can satisfy (SupportedEndpoints);
//   - tell upstream code which identifier the running engine expects in the
//     request body's "model" field (ServedModelName);
//   - declare a typed parameter / config schema for UI and validation.
//
// Plugins live in subpackages under internal/backend/plugins/<id>/ and
// register themselves at process start via init() blank imports from
// cmd/shepherd/plugins.go.
//
// All plugin-typed parameters travel through LoadRequest.Params as an opaque
// interface satisfied by per-plugin marker types. The handler / model manager
// stays plugin-agnostic; only the plugin's BuildStartConfig type-asserts the
// concrete type back.
package backend

// ID is the canonical identifier of a plugin (e.g. "llamacpp", "vllm",
// "vllmomni"). It is also the value persisted in the database column and
// returned in API responses, so it must remain stable across releases.
type ID string

// String makes ID printable; the value is already a string under the hood.
func (i ID) String() string { return string(i) }

// Plugin is the contract every backend plugin must implement.
//
// Method groups (top to bottom):
//   - identity:          ID, DisplayName
//   - lifecycle:         Discover, BuildStartConfig
//   - runtime behaviour: IsLoadComplete, CheckHealth, SupportsModel, SupportedEndpoints
//   - request shaping:   ServedModelName
//   - schema / config:   ConfigSchema, ParamSchema, DecodeParams, ValidateParams
//
// All methods are expected to be safe for concurrent use; plugins typically
// hold no mutable state.
type Plugin interface {
	// -- identity --

	// ID returns the canonical, lowercase, [a-z0-9_] identifier of this plugin.
	// Used as DB key and map key throughout the system.
	ID() ID

	// DisplayName returns a human-readable name (e.g. "vLLM-Omni"), suitable
	// for UI and API responses.
	DisplayName() string

	// -- lifecycle --

	// Discover validates that the engine binary referenced by cfg is present
	// and usable, optionally probing its version. The returned Info.Available
	// flag indicates whether the backend can actually serve models.
	//
	// A nil error with Available=false means "not installed" — callers should
	// surface that as a configuration problem, not a hard failure.
	Discover(cfg *Config) (*Info, error)

	// BuildStartConfig produces the StartConfig that drives the process
	// supervisor. Plugins type-assert req.Params to their own concrete type
	// here; receiving a wrong type must yield ErrParamTypeMismatch.
	BuildStartConfig(info *Info, req *LoadRequest) (*StartConfig, error)

	// -- runtime --

	// IsLoadComplete returns true when the given line of stdout/stderr
	// indicates the backend has finished loading and is accepting requests.
	IsLoadComplete(stdoutLine string) bool

	// CheckHealth probes the spawned process at the given local port and
	// returns whether it is responsive.
	CheckHealth(port int) (*HealthResult, error)

	// SupportsModel decides whether this plugin can serve the given model
	// path. Used by the resolver to filter candidates.
	SupportsModel(modelPath string) bool

	// SupportedEndpoints lists the OpenAI-compatible API endpoints this
	// plugin's runtime can satisfy (e.g. "/v1/chat/completions",
	// "/v1/audio/speech"). Mapped to true if supported.
	SupportedEndpoints() map[string]bool

	// -- request shaping --

	// ServedModelName returns the identifier the running backend expects in
	// the JSON body's "model" field. llama.cpp recognises the model's friendly
	// name; vLLM-family backends recognise the on-disk path. Returning the
	// path or name is plugin-specific — this method exists so the compat
	// layer does not need to switch on plugin ID.
	ServedModelName(m ModelRef) string

	// -- schema / config --

	// ConfigSchema describes the YAML keys this plugin accepts under
	// `backends.<id>:` in server.config.yaml. Used by the config loader to
	// decode the raw YAML node into the plugin's typed config.
	ConfigSchema() ConfigSchema

	// ParamSchema describes the per-load parameters the user can tune (UI
	// surface and validation).
	ParamSchema() ParamSchema

	// DecodeParams converts an untyped JSON parameter map (as received from
	// the API) into the plugin's concrete Params type.
	DecodeParams(raw RawParams) (Params, error)

	// ValidateParams checks an untyped parameter map for type errors,
	// out-of-range values, and unknown keys. Used by handlers before
	// scheduling a load.
	ValidateParams(raw RawParams) ValidationResult
}

// ModelRef is the minimal model description handed to ServedModelName.
type ModelRef struct {
	ID   string
	Name string
	Path string
}

// Params is the marker interface implemented by every plugin's typed
// parameter struct. Plugin subpackages embed ParamsBase to satisfy it.
type Params interface {
	pluginParams()
}

// ParamsBase is embedded by plugin Params structs to satisfy the Params
// interface across package boundaries.
type ParamsBase struct{}

func (ParamsBase) pluginParams() {}

// RawParams is the on-the-wire JSON shape of plugin parameters: an untyped
// map[string]any. Decoded into a plugin's typed Params via Plugin.DecodeParams.
type RawParams map[string]any

// LoadRequest is the plugin-agnostic input to BuildStartConfig. Only fields
// truly common to every engine live here; anything engine-specific (GPU
// layers, threads, ctx size, speculative decoding) belongs on Params.
type LoadRequest struct {
	// ModelPath is the on-disk path to the model artifact.
	ModelPath string

	// Port is the local TCP port the spawned process binds to.
	Port int

	// Devices is an opaque list of device selectors; each plugin parses its
	// own expected format.
	Devices []string

	// BindHost is the listen address (e.g. "0.0.0.0", "127.0.0.1").
	// Required: callers must resolve it from plugin Config before validating.
	BindHost string

	// Params carries plugin-specific parameters. Plugins type-assert inside
	// BuildStartConfig.
	Params Params

	// EnvVars are model-level env overrides on top of plugin Config.Env.
	EnvVars []string

	// ExtraArgs is a raw passthrough appended to the command line.
	ExtraArgs string
}

// Validate enforces invariants common to every plugin. Plugin-specific
// invariants live on the plugin's typed Params struct.
func (r *LoadRequest) Validate() error {
	if r == nil {
		return ErrInvalidLoadRequest("LoadRequest is nil")
	}
	if r.ModelPath == "" {
		return ErrInvalidLoadRequest("ModelPath is required")
	}
	if r.Port <= 0 {
		return ErrInvalidLoadRequest("Port must be positive")
	}
	if r.BindHost == "" {
		return ErrInvalidLoadRequest("BindHost is required")
	}
	return nil
}

// Info is what Discover returns: enough metadata to call BuildStartConfig.
type Info struct {
	ID              ID
	DisplayName     string
	BinPath         string
	Version         string
	Available       bool
	CondaEnv        string
	CondaPath       string
	GlobalExtraArgs string
}

// Config is the plugin-agnostic configuration stored in the Registry.
// Plugins read their slice via Raw (untyped YAML) or Decoded (typed value
// from PluginConfigDecoder.DecodeConfig).
type Config struct {
	ID          ID
	DisplayName string
	BinPaths    []string
	BindHost    string

	// Raw is the unmarshalled YAML node under `backends.<id>:`.
	Raw map[string]any

	// Decoded is the typed value from PluginConfigDecoder.DecodeConfig,
	// or nil if the plugin does not implement that interface.
	Decoded any
}

// StartConfig is what the process supervisor receives. The supervisor reads
// CommandSpec.Binary + Args directly (no shell interpretation); use
// CommandSpec.CommandLine() to render a quoted string for logs or previews.
type StartConfig struct {
	CommandSpec       *CommandSpec
	BinPath           string
	PluginID          ID
	SkipLDLibraryPath bool
	CondaPath         string
	EnvVars           []string
}

// HealthResult is returned by Plugin.CheckHealth.
type HealthResult struct {
	Healthy bool
	Body    string
}

// CapabilityHint is supplied to Registry.Resolve to bias selection toward
// multimodal-capable plugins when the model declares such a need.
type CapabilityHint struct {
	TTS             bool
	ASR             bool
	ImageGeneration bool
}

// NeedsMultimodal reports whether any audio / image capability is requested.
func (h *CapabilityHint) NeedsMultimodal() bool {
	if h == nil {
		return false
	}
	return h.TTS || h.ASR || h.ImageGeneration
}

// ConfigField describes a single key under `backends.<id>:` for documentation
// and (eventually) UI rendering.
type ConfigField struct {
	Name        string
	Type        ParamType
	Description string
	Default     any
	Required    bool
}

// ConfigSchema is the union of ConfigField values a plugin accepts.
type ConfigSchema struct {
	PluginID ID
	Fields   []ConfigField
}

// ParamType enumerates the primitive value types supported by ParamDef and
// ConfigField. Strings is the array-of-strings shape (e.g. samplers list).
type ParamType string

const (
	ParamTypeInt     ParamType = "int"
	ParamTypeFloat   ParamType = "float"
	ParamTypeString  ParamType = "string"
	ParamTypeBool    ParamType = "bool"
	ParamTypeStrings ParamType = "strings"
)

// ParamDef describes a single tunable parameter for a plugin.
//
// JSONName is the key carried over the wire; Flag is the corresponding
// command-line flag the plugin emits when constructing the start command.
// Either may be empty if the parameter is not surfaced in that domain.
type ParamDef struct {
	Name         string
	JSONName     string
	Flag         string
	Type         ParamType
	Group        string
	Description  string
	Default      any
	Min          *float64
	Max          *float64
	Options      []any
	Advanced     bool
	SinceVersion string
}

// ParamSchema is what Plugin.ParamSchema() returns: the full set of ParamDef
// entries for one plugin.
type ParamSchema struct {
	PluginID ID
	Params   []ParamDef
}

// ValidationResult is the return shape of Plugin.ValidateParams.
type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}
