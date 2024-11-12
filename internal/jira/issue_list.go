package jira

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/andygrunwald/go-jira/v2/cloud"
)

// IssueList holds a list of Jira issues and provides methods for displaying them.
type IssueList struct {
	Issues     []cloud.Issue
	MaxResults int
	Total      int
}

// NewIssueList initializes a new IssueList with a given slice of issues.
func NewIssueList(issues []cloud.Issue, max, total int) *IssueList {
	return &IssueList{Issues: issues, MaxResults: max, Total: total}
}

// Count returns the number of issues in the list.
func (il *IssueList) Count() int {
	return len(il.Issues)
}

// Print displays the issues on the console.
func (il *IssueList) Print() {
	var keyWidth, statusWidth int

	// Check if there are issues
	if len(il.Issues) == 0 {
		fmt.Println("No results found.")
		return
	}

	// Determine the maximum width for issue keys and statuses
	for _, issue := range il.Issues {
		if len(issue.Key) > keyWidth {
			keyWidth = len(issue.Key)
		}
		if len(issue.Fields.Status.Name) > statusWidth {
			statusWidth = len(issue.Fields.Status.Name)
		}
	}

	// Print each issue with proper formatting
	for _, issue := range il.Issues {
		// Default color for issue key
		issueKeyColor := "\033[1;34m"

		// Custom color based on the issue's status category
		switch issue.Fields.Status.StatusCategory.Key {
		case "new":
			issueKeyColor = "\033[1;37m"
		case "done":
			issueKeyColor = "\033[1;32m"
		}

		// Set assignee to "Unassigned" if not present
		assigneeName := "Unassigned"
		if issue.Fields.Assignee != nil {
			assigneeName = issue.Fields.Assignee.DisplayName
		}

		// Format the updated time
		updatedTime := time.Time(issue.Fields.Updated).Format("02-01-2006")

		// Print formatted issue details
		fmt.Printf(
			"[%s%-*s\033[0m][%s%-*s\033[0m][%s][\033[34m%s\033[0m](\033[33m%s\033[0m)\033[1;37m %s\033[0m\n",
			issueKeyColor,
			keyWidth,
			issue.Key,
			issueKeyColor,
			statusWidth,
			issue.Fields.Status.Name,
			updatedTime,
			assigneeName,
			issue.Fields.Project.Name,
			issue.Fields.Summary,
		)
	}

	if il.Total > il.MaxResults {
		fmt.Printf("\033[1;32m * \033[1;31mDisplaying first %d of %d results\033[0m\n", il.MaxResults, il.Total)
	}
}

// ToJSON converts the IssueList to a JSON representation.
func (il *IssueList) ToJSON() (string, error) {
	data, err := json.MarshalIndent(il, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error converting issues to JSON: %w", err)
	}
	return string(data), nil
}
