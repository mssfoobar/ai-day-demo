package aohhttp

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var ErrInvalidPageNumber = errors.New("invalid page number")
var ErrInvalidPageSize = errors.New("invalid page size")
var ErrInvalidSort = errors.New("invalid sort")

type PageRequest struct {
	Number int
	Size   int
	Sorts  Sorts
}

type PageResponse struct {
	Number       int      `json:"number"`
	Size         int      `json:"size"`
	TotalRecords int      `json:"total_records"`
	Count        int      `json:"count"`
	Sort         []string `json:"sort"`
}

type Sort struct {
	Column    string
	Direction string
}

type Sorts []Sort

// String returns a string representation of the sorts in the format "column1 direction1, column2 direction2, ..."
func (s Sorts) String() string {
	var l []string
	for _, v := range s {
		l = append(l, fmt.Sprintf("%s %s", v.Column, v.Direction))
	}
	return strings.Join(l, ", ")
}

func NewPageRequest() PageRequest {
	return PageRequest{
		Number: 1,
		Size:   10,
		Sorts:  make([]Sort, 0),
	}
}

// GetQueryPagination extracts the query parameters from the request and returns a PageRequest struct
func GetQueryPagination(r *http.Request) (*PageRequest, error) {
	page := NewPageRequest()
	v := r.URL.Query()

	num := v.Get("page")
	if num != "" {
		iNum, err := strconv.Atoi(num)
		if err != nil || iNum < 1 {
			return nil, ErrInvalidPageNumber
		}
		page.Number = iNum
	}

	size := v.Get("size")
	if size != "" {
		iSize, err := strconv.Atoi(size)
		if err != nil || iSize < 1 {
			return nil, ErrInvalidPageSize
		}
		page.Size = iSize
	}

	sorts := v["sort"]
	for _, sort := range sorts {
		s, err := parseSort(sort)
		if err != nil {
			return nil, err
		}
		page.Sorts = append(page.Sorts, *s)
	}

	return &page, nil
}

func parseSort(s string) (*Sort, error) {
	var sort Sort
	strs := strings.Split(s, ",")
	switch len(strs) {
	case 1:
		if strs[0] != "" {
			sort.Column = strs[0]
			sort.Direction = "desc"
		} else {
			return nil, ErrInvalidSort
		}
	case 2:
		sort.Column = strs[0]
		smallStr := strings.ToLower(strs[1])
		if smallStr != "asc" && smallStr != "desc" {
			return nil, ErrInvalidSort
		}
		sort.Direction = smallStr
	default:
		return nil, ErrInvalidSort
	}
	return &sort, nil
}
