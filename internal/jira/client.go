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
