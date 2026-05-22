package gguf

import (
	"github.com/simonxluo/Shepherd/internal/comm/utils"
)

func ReadMetadata(path string) (*Metadata, error) {
	parser, err := NewParser(path)
	if err != nil {
		return nil, err
	}
	defer utils.CloseQuietly(parser)

	return parser.GetMetadata()
}
