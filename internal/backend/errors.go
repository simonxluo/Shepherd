package backend

import (
	"errors"
	"fmt"
)

// Sentinel errors. Callers should use errors.Is to test rather than comparing
// strings.
var (
	// ErrPluginNotFound indicates the requested plugin ID has not been
	// registered. Returned by Registry.Get and Registry.Resolve.
	ErrPluginNotFound = errors.New("plugin not found")

	// ErrPluginNotConfigured indicates the plugin is registered but its
	// Config has not been populated (e.g. the operator did not enable it
	// in server.config.yaml).
	ErrPluginNotConfigured = errors.New("plugin not configured")

	// ErrParamTypeMismatch indicates LoadRequest.Params holds a value whose
	// concrete type the receiving plugin does not recognise. Should never
	// happen if callers go through Plugin.DecodeParams.
	ErrParamTypeMismatch = errors.New("plugin param type mismatch")

	// ErrNoSuitableBackend indicates Resolve found no plugin willing to
	// handle the given model.
	ErrNoSuitableBackend = errors.New("no suitable backend for model")
)

// ErrInvalidLoadRequest wraps a validation message for LoadRequest.Validate.
// Returned via the package-level helper so callers can do:
//
//	if err := req.Validate(); err != nil { ... }
//
// without importing fmt for the sentinel comparison.
func ErrInvalidLoadRequest(msg string) error {
	return fmt.Errorf("invalid load request: %s", msg)
}

// PluginNotFoundError formats a not-found message that includes the plugin ID.
// Wraps ErrPluginNotFound so errors.Is keeps working.
func PluginNotFoundError(id ID) error {
	return fmt.Errorf("%w: %s", ErrPluginNotFound, id)
}

// PluginNotConfiguredError mirrors PluginNotFoundError for the "not
// configured" case.
func PluginNotConfiguredError(id ID) error {
	return fmt.Errorf("%w: %s", ErrPluginNotConfigured, id)
}

// NoSuitableBackendError reports the model path that no plugin would accept.
func NoSuitableBackendError(modelPath string) error {
	return fmt.Errorf("%w: %s", ErrNoSuitableBackend, modelPath)
}
