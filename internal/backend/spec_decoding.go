package backend

// SpecDecodingParams holds speculative decoding configuration shared by the
// API layer and the llamacpp plugin. SpecDraftModelID is the API-side
// reference (resolved to SpecDraftModelPath by the handler before the load
// request reaches the plugin); SpecDraftModelPath is the on-disk path the
// plugin actually consumes.
type SpecDecodingParams struct {
	SpecType string `json:"specType"`

	// draft / eagle3
	SpecDraftModelID   string  `json:"specDraftModelId"`
	SpecDraftModelPath string  `json:"-"`
	SpecDraftNMax      int     `json:"specDraftNMax"`
	SpecDraftNMin      int     `json:"specDraftNMin"`
	SpecDraftPSplit    float64 `json:"specDraftPSplit"`
	SpecDraftPMin      float64 `json:"specDraftPMin"`
	SpecDraftCtxSize   int     `json:"specDraftCtxSize"`
	SpecDraftNGL       int     `json:"specDraftNgl"`
	SpecDraftDevice    string  `json:"specDraftDevice"`

	// ngram-mod
	SpecNgramModNMin   int `json:"specNgramModNMin"`
	SpecNgramModNMax   int `json:"specNgramModNMax"`
	SpecNgramModNMatch int `json:"specNgramModNMatch"`

	// ngram-simple
	SpecNgramSimpleSizeN   int `json:"specNgramSimpleSizeN"`
	SpecNgramSimpleSizeM   int `json:"specNgramSimpleSizeM"`
	SpecNgramSimpleMinHits int `json:"specNgramSimpleMinHits"`

	// ngram-map-k
	SpecNgramMapKSizeN   int `json:"specNgramMapKSizeN"`
	SpecNgramMapKSizeM   int `json:"specNgramMapKSizeM"`
	SpecNgramMapKMinHits int `json:"specNgramMapKMinHits"`

	// ngram-map-k4v
	SpecNgramMapK4VSizeN   int `json:"specNgramMapK4VSizeN"`
	SpecNgramMapK4VSizeM   int `json:"specNgramMapK4VSizeM"`
	SpecNgramMapK4VMinHits int `json:"specNgramMapK4VMinHits"`

	// ngram-cache
	LookupCacheStatic  string `json:"lookupCacheStatic"`
	LookupCacheDynamic string `json:"lookupCacheDynamic"`
}
