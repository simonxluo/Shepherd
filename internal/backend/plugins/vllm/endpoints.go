package vllm

// baseEndpoints returns the common API endpoint set.
func baseEndpoints() map[string]bool {
	return map[string]bool{
		"/v1/chat/completions": true,
		"/v1/completions":      true,
		"/v1/models":           true,
		"/v1/embeddings":       true,
	}
}

// endpointsWithoutAudio returns the base set with audio endpoints disabled.
func endpointsWithoutAudio() map[string]bool {
	ep := baseEndpoints()
	ep["/v1/audio/speech"] = false
	ep["/v1/audio/voices"] = false
	ep["/v1/audio/transcriptions"] = false
	ep["/v1/audio/translations"] = false
	ep["/v1/audio/music"] = false
	return ep
}

// EndpointsWithAudio returns the base set with all audio endpoints enabled.
func EndpointsWithAudio() map[string]bool {
	ep := baseEndpoints()
	ep["/v1/audio/speech"] = true
	ep["/v1/audio/voices"] = true
	ep["/v1/audio/transcriptions"] = true
	ep["/v1/audio/translations"] = true
	ep["/v1/audio/music"] = true
	return ep
}
