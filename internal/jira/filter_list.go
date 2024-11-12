package jira

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/andygrunwald/go-jira/v2/cloud"
)

// FilterList holds a list of Jira filters and provides methods for displaying them.
type FilterList struct {
	Filters    []cloud.FiltersListItem
	MaxResults int
	Total      int
}

// NewFilterList initializes a new FilterList with a given slice of filters.
func NewFilterList(filters []cloud.FiltersListItem, max, total int) *FilterList {
	return &FilterList{Filters: filters, MaxResults: max, Total: total}
}

// Count returns the number of filters in the list.
func (fl *FilterList) Count() int {
	return len(fl.Filters)
}

// Print displays the filters on the console.
func (fl *FilterList) Print() {
	if len(fl.Filters) == 0 {
		fmt.Println("No filters found.")
		return
	}

	// Sort the filters by their Name
	sort.Slice(fl.Filters, func(i, j int) bool {
		return fl.Filters[i].Name < fl.Filters[j].Name
	})

	for _, filter := range fl.Filters {
		fmt.Printf("\033[1;34m%s\033[0m: \033[33m%s\033[0m\n", filter.ID, filter.Name)
	}

	if fl.Total > fl.MaxResults {
		fmt.Printf("\033[1;32m * \033[1;31mDisplaying first %d of %d filters\033[0m\n", fl.MaxResults, fl.Total)
	}
}

// ToJSON converts the FilterList to a JSON representation.
func (fl *FilterList) ToJSON() (string, error) {
	data, err := json.MarshalIndent(fl, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error converting filters to JSON: %w", err)
	}
	return string(data), nil
}
