package cursor

// PageMeta makes traversal boundaries and opaque cursors explicit.
type PageMeta struct {
	// HasPrevious is an application-supplied boundary result, not computed by BuildPage.
	HasPrevious bool `json:"has_previous"`
	// HasNext is an application-supplied boundary result, not computed by BuildPage.
	HasNext bool `json:"has_next"`
	// PreviousCursor is encoded from the first item when HasPrevious is true and items are
	// nonempty.
	PreviousCursor string `json:"previous_cursor,omitempty"`
	// NextCursor is encoded from the last item when HasNext is true and items are
	// nonempty.
	NextCursor string `json:"next_cursor,omitempty"`
}

// Page is a stable response envelope whose items remain in canonical sort
// order for both forward and backward requests, as supplied by the application.
// BuildPage does not sort or reverse results.
type Page[T any] struct {
	// Items is a shallow slice snapshot in application-provided canonical order.
	Items []T `json:"items"`
	// Meta records application-proven boundaries and encoded tokens.
	Meta PageMeta `json:"page"`
}

// PageEncoder encodes one boundary item for a traversal direction.
type PageEncoder[T any] func(item T, direction Direction) (string, error)

// BuildPage snapshots canonical-order items and encodes only available
// boundaries. Storage adapters determine hasPrevious and hasNext explicitly.
// The slice copy is shallow. encode runs synchronously, first previous then next,
// and must be non-nil when a nonempty boundary needs encoding. On encoder failure
// the zero Page and callback error are returned; prior callback effects remain.
func BuildPage[T any](items []T, hasPrevious, hasNext bool, encode PageEncoder[T]) (Page[T], error) {
	page := Page[T]{Items: append([]T(nil), items...), Meta: PageMeta{
		HasPrevious: hasPrevious, HasNext: hasNext,
	}}
	if len(items) == 0 {
		return page, nil
	}
	if hasPrevious {
		cursor, err := encode(items[0], Backward)
		if err != nil {
			return Page[T]{}, err
		}
		page.Meta.PreviousCursor = cursor
	}
	if hasNext {
		cursor, err := encode(items[len(items)-1], Forward)
		if err != nil {
			return Page[T]{}, err
		}
		page.Meta.NextCursor = cursor
	}
	return page, nil
}
