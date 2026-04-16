package api

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

type paginationResponse struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

type pagedItemsResponse[T any] struct {
	Items      []T                `json:"items"`
	Pagination paginationResponse `json:"pagination"`
}

type paginationRequest struct {
	Page    int
	PerPage int
}

func parsePagination(r *http.Request) (paginationRequest, error) {
	query := r.URL.Query()
	page := 1
	perPage := defaultPageSize

	if rawPage := strings.TrimSpace(query.Get("page")); rawPage != "" {
		parsed, err := strconv.Atoi(rawPage)
		if err != nil || parsed <= 0 {
			return paginationRequest{}, fmt.Errorf("page must be a positive integer")
		}
		page = parsed
	}

	if rawPerPage := strings.TrimSpace(query.Get("per_page")); rawPerPage != "" {
		parsed, err := strconv.Atoi(rawPerPage)
		if err != nil || parsed <= 0 {
			return paginationRequest{}, fmt.Errorf("per_page must be a positive integer")
		}
		if parsed > maxPageSize {
			parsed = maxPageSize
		}
		perPage = parsed
	}

	return paginationRequest{Page: page, PerPage: perPage}, nil
}

func buildPagination(total, page, perPage int) paginationResponse {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = defaultPageSize
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}

	return paginationResponse{
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}
}

func slicePage[T any](items []T, page, perPage int) ([]T, paginationResponse) {
	pagination := buildPagination(len(items), page, perPage)
	if len(items) == 0 {
		return []T{}, pagination
	}

	start := (pagination.Page - 1) * pagination.PerPage
	if start >= len(items) {
		return []T{}, pagination
	}
	end := start + pagination.PerPage
	if end > len(items) {
		end = len(items)
	}
	return append([]T(nil), items[start:end]...), pagination
}
