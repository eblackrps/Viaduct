package api

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParsePagination_Defaults_Expected(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/inventory", nil)

	paging, err := parsePagination(request)
	if err != nil {
		t.Fatalf("parsePagination() error = %v", err)
	}
	if paging.Page != 1 || paging.PerPage != defaultPageSize {
		t.Fatalf("parsePagination() = %#v, want page 1 per_page %d", paging, defaultPageSize)
	}
}

func TestParsePagination_PerPageCap_Expected(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/inventory?per_page=999999", nil)

	paging, err := parsePagination(request)
	if err != nil {
		t.Fatalf("parsePagination() error = %v", err)
	}
	if paging.PerPage != maxPageSize {
		t.Fatalf("PerPage = %d, want cap %d", paging.PerPage, maxPageSize)
	}
}

func TestParsePagination_InvalidPageRejected_Expected(t *testing.T) {
	for _, raw := range []string{"0", "-1", "not-an-int"} {
		t.Run(raw, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/api/v1/inventory?page="+raw, nil)

			_, err := parsePagination(request)
			if err == nil {
				t.Fatal("parsePagination() error = nil, want invalid page error")
			}
			if !strings.Contains(err.Error(), "page") {
				t.Fatalf("parsePagination() error = %q, want page guidance", err)
			}
		})
	}
}

func TestParsePagination_HugePageRejected_Expected(t *testing.T) {
	request := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/inventory?page=%d", maxPageNumber+1), nil)

	_, err := parsePagination(request)
	if err == nil {
		t.Fatal("parsePagination() error = nil, want huge page rejection")
	}
	if !strings.Contains(err.Error(), "no greater than") {
		t.Fatalf("parsePagination() error = %q, want max-page guidance", err)
	}
}

func TestSlicePage_HugePageDoesNotPanic_Expected(t *testing.T) {
	items := []int{1, 2, 3}

	page, pagination := slicePage(items, int(^uint(0)>>1), maxPageSize)
	if len(page) != 0 {
		t.Fatalf("len(page) = %d, want empty result", len(page))
	}
	if pagination.Total != len(items) {
		t.Fatalf("pagination.Total = %d, want %d", pagination.Total, len(items))
	}
}

func TestSlicePage_LargePerPageAndEmptySliceSafe_Expected(t *testing.T) {
	empty, pagination := slicePage([]int{}, 1, int(^uint(0)>>1))
	if len(empty) != 0 {
		t.Fatalf("len(empty) = %d, want 0", len(empty))
	}
	if pagination.Total != 0 || pagination.Page != 1 {
		t.Fatalf("pagination = %#v, want empty page metadata", pagination)
	}

	page, pagination := slicePage([]int{1, 2, 3}, 1, int(^uint(0)>>1))
	if len(page) != 3 {
		t.Fatalf("len(page) = %d, want 3", len(page))
	}
	if pagination.PerPage <= 0 {
		t.Fatalf("pagination.PerPage = %d, want positive value", pagination.PerPage)
	}
}
