package jira

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/andygrunwald/go-jira/v2/cloud"
)

// ProjectList holds a list of Jira projects and provides methods for displaying them.
type ProjectList struct {
	Projects   cloud.ProjectList
	MaxResults int
	Total      int
}

// NewProjectList initializes a new ProjectList with a given slice of projects.
func NewProjectList(projects *cloud.ProjectList, max, total int) *ProjectList {
	return &ProjectList{Projects: *projects, MaxResults: max, Total: total}
}

// Count returns the number of projects in the list.
func (pl *ProjectList) Count() int {
	return len(pl.Projects)
}

// Print displays the projects on the console.
func (pl *ProjectList) Print() {
	if len(pl.Projects) == 0 {
		fmt.Println("No projects found.")
		return
	}

	// Sort the projects by their Key
	sort.Slice(pl.Projects, func(i, j int) bool {
		return (pl.Projects)[i].Key < (pl.Projects)[j].Key
	})

	for _, project := range pl.Projects {
		fmt.Printf("\033[1;34m%s\033[0m: \033[33m%s\033[0m\n", project.Key, project.Name)
	}

	if pl.Total > pl.MaxResults {
		fmt.Printf("\033[1;32m * \033[1;31mDisplaying first %d of %d projects\033[0m\n", pl.MaxResults, pl.Total)
	}
}

// ToJSON converts the ProjectList to a JSON representation.
func (pl *ProjectList) ToJSON() (string, error) {
	data, err := json.MarshalIndent(pl, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error converting projects to JSON: %w", err)
	}
	return string(data), nil
}
