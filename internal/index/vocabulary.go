package index

import (
	"context"
)

type SearchVocabularyOptions struct {
	Limit                     int
	MaxDocumentFrequencyRatio float64
	MinTermRunes              int
	MaxTermRunes              int
	MaxWords                  int
}

func SearchVocabularyFromStore(ctx context.Context, store Store, opts SearchVocabularyOptions) ([]string, error) {
	return store.SearchVocabulary(ctx, opts)
}
