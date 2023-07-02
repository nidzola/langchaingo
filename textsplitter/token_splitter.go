package textsplitter

import (
	"fmt"
	"github.com/pkoukk/tiktoken-go"
)

const (
	_defaultTokenModelName = "gpt-3.5-turbo"
	_defaultTokenEncoding  = "cl100k_base"
)

// TokenSplitter is a text splitter that will split texts by tokens.
type TokenSplitter struct {
	ChunkSize         int
	ChunkOverlap      int
	ModelName         string
	EncodingName      string
	AllowedSpecial    []string
	DisallowedSpecial []string
}

func NewTokenSplitter() TokenSplitter {
	return TokenSplitter{
		ChunkSize:         512,
		ChunkOverlap:      100,
		ModelName:         _defaultTokenModelName,
		EncodingName:      _defaultTokenEncoding,
		AllowedSpecial:    []string{},
		DisallowedSpecial: []string{"all"},
	}
}

// SplitText splits a text into multiple text.
func (s TokenSplitter) SplitText(text string) ([]string, error) {
	// Get the tokenizer
	var tk *tiktoken.Tiktoken
	var err error
	if s.ModelName != "" {
		tk, err = tiktoken.EncodingForModel(s.ModelName)
	} else if s.EncodingName != "" {
		tk, err = tiktoken.GetEncoding(s.EncodingName)
	} else {
		err = fmt.Errorf("must have either model name or encoding name")
	}
	if err != nil {
		return nil, fmt.Errorf("tiktoken.GetEncoding: %w", err)
	}
	texts := s.splitText(text, tk)

	return texts, nil
}

func (s TokenSplitter) splitText(text string, tk *tiktoken.Tiktoken) []string {
	splits := make([]string, 0)
	inputIds := tk.Encode(text, s.AllowedSpecial, s.DisallowedSpecial)

	startIdx := 0
	curIdx := len(inputIds)
	if startIdx+s.ChunkSize < curIdx {
		curIdx = startIdx + s.ChunkSize
	}
	for startIdx < len(inputIds) {
		chunkIds := inputIds[startIdx:curIdx]
		splits = append(splits, tk.Decode(chunkIds))
		startIdx += s.ChunkSize - s.ChunkOverlap
		curIdx = startIdx + s.ChunkSize
		if curIdx > len(inputIds) {
			curIdx = len(inputIds)
		}
	}
	return splits
}
