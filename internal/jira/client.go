package jira

import (
	"context"
	"fmt"
	"net/http"

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
func (c *Client) SearchIssuesWithPagination(jql string, maxResults int) (*IssueList, error) {
	var allIssues []cloud.Issue
	startAt := 0
	totalFetched := 0
	pageSize := 50
	total := 0

	for {
		// Fetch issues in pages of size pageSize
		issueList, err := c.SearchIssues(jql, startAt, pageSize)
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
func (c *Client) SearchIssues(jql string, start, limit int) (*IssueList, error) {
	searchOptions := &cloud.SearchOptions{
		StartAt:    start,
		MaxResults: limit,
	}

	issues, response, err := c.apiClient.Issue.Search(context.Background(), jql, searchOptions)
	if err != nil {
		return nil, fmt.Errorf("error executing JQL query: %w", err)
	}

	// Create an IssueList
	return NewIssueList(issues, response.MaxResults, response.Total), nil
}

// SearchIssuesByFilter retrieves issues using a pre-existing saved filter by its ID and returns an IssueList with pagination.
func (c *Client) SearchIssuesByFilter(filterID string, limit int) (*IssueList, error) {
	// Create the JQL query with the saved filter
	jql := fmt.Sprintf("filter=%s", filterID)

	// Initialize pagination variables
	var allIssues []cloud.Issue
	startAt := 0
	fetchLimit := 100 // Set a limit for each page of results (maximum Jira allows is 1000)

	// Iterate over pages of issues
	for {
		// Create the search options with pagination
		searchOptions := &cloud.SearchOptions{
			StartAt:    startAt,
			MaxResults: fetchLimit,
		}

		// Execute the search with the saved filter
		issues, response, err := c.apiClient.Issue.Search(context.Background(), jql, searchOptions)
		if err != nil {
			return nil, fmt.Errorf("error executing JQL query with filter %s: %w", filterID, err)
		}

		// Append issues to the list
		allIssues = append(allIssues, issues...)

		// Check if there are more pages to fetch
		if len(allIssues) >= limit || len(allIssues) >= response.Total {
			break
		}

		// Update the starting point for the next page
		startAt += fetchLimit
	}

	// Return the IssueList with the fetched issues and pagination details
	return NewIssueList(allIssues, len(allIssues), len(allIssues)), nil
}

// GetAllProjects retrieves all visible Jira projects.
func (c *Client) GetAllProjects() (*ProjectList, error) {
	// Fetch the list of projects using the Jira API
	projectList, response, err := c.apiClient.Project.GetAll(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching projects: %w", err)
	}

	if response.Total > response.MaxResults {
		fmt.Printf("\033[1;31mDisplaying first %d results of %d\033[0m\n", response.MaxResults, response.Total)
	}

	// Return the list of projects (accessing the Projects field from the ProjectList)
	return NewProjectList(projectList, response.MaxResults, response.Total), nil
}

// GetAllUsers retrieves all visible Jira users, with pagination.
func (c *Client) GetAllUsers() (*UserList, error) {
	var allUsers []cloud.User
	startAt := 0
	maxResults := 1000 // Maximum number of results per request

	for {
		// Prepare the request with pagination
		req, err := c.apiClient.NewRequest(context.Background(), http.MethodGet, fmt.Sprintf("/rest/api/2/users?startAt=%d&maxResults=%d", startAt, maxResults), nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}

		// Store users in this batch
		users := []cloud.User{}
		resp, err := c.apiClient.Do(req, &users)
		if err != nil {
			return nil, cloud.NewJiraError(resp, err)
		}

		// Add the batch of users to the full list
		allUsers = append(allUsers, users...)

		// If fewer users than the maxResults were returned, we've fetched all users
		if len(users) < maxResults {
			break
		}

		// Otherwise, increment the startAt parameter to fetch the next page
		startAt += maxResults
	}

	return NewUserList(allUsers, maxResults, len(allUsers)), nil
}

// GetAllFilters retrieves all saved filters from Jira using the apiClient.
func (c *Client) GetAllFilters() (*FilterList, error) {
	// Use the GetList method from apiClient.Filter to retrieve filters
	filters, response, err := c.apiClient.Filter.Search(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching filters: %w", err)
	}

	return NewFilterList(filters.Values, response.MaxResults, response.Total), nil
}

// GetFilter retrieves an existing Filter from Jira using the apiClient.
func (c *Client) GetFilter(id int) (*cloud.Filter, error) {
	filter, _, err := c.apiClient.Filter.Get(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return filter, nil
}
