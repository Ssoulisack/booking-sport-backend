package services

import (
	"backend/internal/core/dto"
	"backend/internal/core/port/output"
	"backend/pkg/response"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

var (
	currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
	slugCleaner     = regexp.MustCompile(`[^a-z0-9]+`)
)

// normalizePage applies the default and maximum page size
func normalizePage(q dto.ListQuery) (page, limit int) {
	page, limit = q.Page, q.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	return page, limit
}

func newMeta(total int64, page, limit int) dto.Meta {
	totalPage := int((total + int64(limit) - 1) / int64(limit))
	return dto.Meta{Total: int(total), Page: page, Limit: limit, TotalPage: totalPage}
}

// newListQuery builds a repository query with the page already normalized
func newListQuery(q dto.ListQuery) (output.ListQuery, int, int) {
	page, limit := normalizePage(q)
	return output.ListQuery{
		Page:    page,
		Limit:   limit,
		Keyword: strings.TrimSpace(q.Keyword),
		Filters: map[string]any{},
	}, page, limit
}

// mapSlice converts each item; the result is never nil so it encodes as [] in JSON
func mapSlice[M, D any](items []M, convert func(M) D) []D {
	out := make([]D, 0, len(items))
	for _, item := range items {
		out = append(out, convert(item))
	}
	return out
}

// pageOf maps a repository page to a PagedResult
func pageOf[M, D any](items []M, total int64, page, limit int, toDTO func(M) D) *dto.PagedResult[D] {
	return &dto.PagedResult[D]{Items: mapSlice(items, toDTO), Meta: newMeta(total, page, limit)}
}

// orBadRequest turns a not-found error from a referenced entity into a 400 with code
func orBadRequest(err error, code string) error {
	if err != nil && isNotFound(err) {
		return response.ErrorBadRequest(code)
	}
	return err
}

func isActive(flag *bool) bool { return flag == nil || *flag }

func validCurrency(c string) bool { return currencyPattern.MatchString(c) }

func slugify(name string) string {
	slug := strings.Trim(slugCleaner.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if slug == "" { // names in non-Latin scripts (Lao, Thai) have no ASCII letters
		slug = "item-" + uuid.NewString()[:8]
	}
	return slug
}

func conflict(code string) error { return response.NewError(http.StatusConflict, code) }

func badRequest(code string) error { return response.ErrorBadRequest(code) }
