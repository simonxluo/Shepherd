package llamacpp

import (
	"fmt"
	"strconv"

	"github.com/simonxluo/Shepherd/internal/backend"
)

// appendSpecDecodingArgs appends speculative decoding CLI flags based on the
// spec type. Returns the arg slice unchanged if spec is nil or "none".
func appendSpecDecodingArgs(args []string, spec *backend.SpecDecodingParams) []string {
	if spec == nil || spec.SpecType == "" || spec.SpecType == "none" {
		return args
	}
	args = append(args, "--spec-type", spec.SpecType)

	switch spec.SpecType {
	case "draft", "eagle3":
		if spec.SpecDraftModelPath != "" {
			args = append(args, "-md", spec.SpecDraftModelPath)
		}
		if spec.SpecDraftNMax > 0 {
			args = append(args, "--spec-draft-n-max", strconv.Itoa(spec.SpecDraftNMax))
		}
		if spec.SpecDraftNMin > 0 {
			args = append(args, "--spec-draft-n-min", strconv.Itoa(spec.SpecDraftNMin))
		}
		if spec.SpecDraftPSplit > 0 {
			args = append(args, "--spec-draft-p-split", fmt.Sprintf("%.2f", spec.SpecDraftPSplit))
		}
		if spec.SpecDraftPMin > 0 {
			args = append(args, "--spec-draft-p-min", fmt.Sprintf("%.2f", spec.SpecDraftPMin))
		}
		if spec.SpecDraftCtxSize > 0 {
			args = append(args, "--spec-draft-ctx-size", strconv.Itoa(spec.SpecDraftCtxSize))
		}
		if spec.SpecDraftNGL > 0 {
			args = append(args, "--spec-draft-ngl", strconv.Itoa(spec.SpecDraftNGL))
		}
		if spec.SpecDraftDevice != "" {
			args = append(args, "--spec-draft-device", spec.SpecDraftDevice)
		}

	case "ngram-simple":
		appendIfPositive(&args, "--spec-ngram-simple-size-n", spec.SpecNgramSimpleSizeN)
		appendIfPositive(&args, "--spec-ngram-simple-size-m", spec.SpecNgramSimpleSizeM)
		appendIfPositive(&args, "--spec-ngram-simple-min-hits", spec.SpecNgramSimpleMinHits)

	case "ngram-mod":
		appendIfPositive(&args, "--spec-ngram-mod-n-min", spec.SpecNgramModNMin)
		appendIfPositive(&args, "--spec-ngram-mod-n-max", spec.SpecNgramModNMax)
		appendIfPositive(&args, "--spec-ngram-mod-n-match", spec.SpecNgramModNMatch)

	case "ngram-map-k":
		appendIfPositive(&args, "--spec-ngram-map-k-size-n", spec.SpecNgramMapKSizeN)
		appendIfPositive(&args, "--spec-ngram-map-k-size-m", spec.SpecNgramMapKSizeM)
		appendIfPositive(&args, "--spec-ngram-map-k-min-hits", spec.SpecNgramMapKMinHits)

	case "ngram-map-k4v":
		appendIfPositive(&args, "--spec-ngram-map-k4v-size-n", spec.SpecNgramMapK4VSizeN)
		appendIfPositive(&args, "--spec-ngram-map-k4v-size-m", spec.SpecNgramMapK4VSizeM)
		appendIfPositive(&args, "--spec-ngram-map-k4v-min-hits", spec.SpecNgramMapK4VMinHits)

	case "ngram-cache":
		if spec.LookupCacheStatic != "" {
			args = append(args, "--lookup-cache-static", spec.LookupCacheStatic)
		}
		if spec.LookupCacheDynamic != "" {
			args = append(args, "--lookup-cache-dynamic", spec.LookupCacheDynamic)
		}
	}
	return args
}

func appendIfPositive(args *[]string, flag string, v int) {
	if v > 0 {
		*args = append(*args, flag, strconv.Itoa(v))
	}
}
