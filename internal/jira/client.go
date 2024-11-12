package jira

import (
	"context"
	"fmt"

	"github.com/andygrunwald/go-jira/v2/cloud"
)

// Client struct encapsulates the Jira API client from go-jira library.
type Client struct {
	apiClient *cloud.Client
}

// NewClient initializes a new Jira client using the go-jira library.
func NewClient(baseURL, apiToken, userEmail string) (*Client, error) {
	if baseURL == "" || apiToken == "" || userEmail == "" {
		return nil, fmt.Errorf("baseURL, apiToken, and userEmail must be provided")
	}

	tp := cloud.BasicAuthTransport{
		Username: userEmail,
		APIToken: apiToken,
	}

	// Initialize the go-jira API client
	apiClient, err := cloud.NewClient(baseURL, tp.Client())
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	return &Client{apiClient: apiClient}, nil
}

// GetIssue retrieves a specific Jira issue by its key.
func (c *Client) GetIssue(ctx context.Context, issueKey string) (*cloud.Issue, error) {
	issue, _, err := c.apiClient.Issue.Get(ctx, issueKey, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching issue %s: %w", issueKey, err)
	}

	return issue, nil
}

// SearchIssuesWithPagination fetches issues based on a JQL query with pagination and applies a result limit.
func (c *Client) SearchIssuesWithPagination(ctx context.Context, jql string, maxResults int) (*IssueList, error) {
	var allIssues []cloud.Issue
	startAt := 0
	totalFetched := 0
	pageSize := 50
	total := 0

	for {
		// Fetch issues in pages of size pageSize
		issueList, err := c.SearchIssues(ctx, jql, startAt, pageSize)
		if err != nil {
			return nil, fmt.Errorf("error fetching issues with pagination: %w", err)
		}

		// Store response calculated result count
		total = issueList.Total

		// Check how many issues to append based on the maxResults limit
		remainingResults := maxResults - totalFetched
		if len(issueList.Issues) > remainingResults {
			issueList.Issues = issueList.Issues[:remainingResults] // Trim to the remaining results
		}

		// Append the fetched issues to the results slice
		allIssues = append(allIssues, issueList.Issues...)
		totalFetched += len(issueList.Issues)

		// If we've fetched enough issues, stop the loop
		if totalFetched >= maxResults {
			break
		}

		// If the number of issues returned is less than the page size, stop the loop
		if len(issueList.Issues) < pageSize {
			break
		}

		// Move to the next page
		startAt += pageSize
	}

	// Return the combined issue list with the total count and max results
	return NewIssueList(allIssues, len(allIssues), total), nil
}

// SearchIssues executes a JQL query to find issues in Jira.
func (c *Client) SearchIssues(ctx context.Context, jql string, start, limit int) (*IssueList, error) {
	searchOptions := &cloud.SearchOptions{
		StartAt:    start,
		MaxResults: limit,
	}

	issues, response, err := c.apiClient.Issue.Search(ctx, jql, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("error executing JQL query: %w", err)
	}

	// Create an IssueList
	return NewIssueList(issues, response.MaxResults, response.Total), nil
}

// GetAllProjects retrieves all visible Jira projects.
func (c *Client) GetAllProjects(ctx context.Context) (cloud.ProjectList, error) {
	// Fetch the list of projects using the Jira API
	projectList, response, err := c.apiClient.Project.GetAll(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching projects: %w", err)
	}

	if response.Total > response.MaxResults {
		fmt.Printf("\033[1;31mDisplaying first %d results of %d\033[0m\n", response.MaxResults, response.Total)
	}

	// Return the list of projects (accessing the Projects field from the ProjectList)
	return *projectList, nil
}
