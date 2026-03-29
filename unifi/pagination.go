package unifi

import (
	"context"
	"fmt"
	"net/http"
)

const defaultPageSize = 200

// PaginatedResponse is the envelope for Integration API paginated responses.
type PaginatedResponse[T any] struct {
	Data       []T `json:"data"`
	Offset     int `json:"offset"`
	Limit      int `json:"limit"`
	Count      int `json:"count"`
	TotalCount int `json:"totalCount"`
}

// ListAll fetches all pages of a paginated Integration API endpoint,
// returning the complete result set transparently.
func ListAll[T any](c *Client, ctx context.Context, app Application, path string) ([]T, error) {
	var all []T
	offset := 0

	for {
		paginatedPath := fmt.Sprintf("%s?offset=%d&limit=%d", path, offset, defaultPageSize)

		var page PaginatedResponse[T]
		if err := c.do(ctx, http.MethodGet, app, paginatedPath, nil, &page); err != nil {
			return nil, err
		}

		all = append(all, page.Data...)

		if offset+page.Count >= page.TotalCount {
			break
		}
		offset += page.Count
	}

	return all, nil
}
