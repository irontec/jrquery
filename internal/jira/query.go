package jira

import (
	"fmt"
	"strings"

	"irontec.com/jquery/config"
)

// QueryBuilder is a helper struct to build flexible JQL queries.
type QueryBuilder struct {
	filters []string
}

// NewQueryBuilder initializes a new QueryBuilder instance.
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{filters: make([]string, 0)}
}

// AddFilter adds a JQL filter to the query.
func (qb *QueryBuilder) AddFilter(field, operator, value string) *QueryBuilder {
	qb.filters = append(qb.filters, fmt.Sprintf("%s %s \"%s\"", field, operator, value))
	return qb
}

// Build generates the final JQL query string.
func (qb *QueryBuilder) Build() string {
	return strings.Join(qb.filters, " AND ")
}

// BuildJQLQuery builds the JQL query from the command line flags
func (qb *QueryBuilder) BuildJQLQuery(flags *config.Flags, searchTerms []string) string {
	var jqlFields []string

	// If custom query is provided, use it directly
	if flags.Query != "" {
		return flags.Query
	}

	// Add project filter if specified
	if flags.Project != "" {
		jqlFields = append(jqlFields, fmt.Sprintf("project = '%s'", flags.Project))
	}

	// Add search conditions if provided
	if len(flags.Search) > 0 {
		var summaryConditions []string
		var descriptionConditions []string
		var commentConditions []string

		// Loop over search terms and build conditions for summary, description, and comments
		for _, term := range searchTerms {
			summaryConditions = append(summaryConditions, fmt.Sprintf("summary ~ '%s'", term))
			descriptionConditions = append(descriptionConditions, fmt.Sprintf("description ~ '%s'", term))
			commentConditions = append(commentConditions, fmt.Sprintf("comment ~ '%s'", term))
		}

		// Combine conditions based on the search terms
		var searchConditions []string
		if len(flags.Search) > 0 {
			searchConditions = append(searchConditions, strings.Join(summaryConditions, " AND "))
		}
		if len(flags.Search) > 1 {
			searchConditions = append(searchConditions, strings.Join(descriptionConditions, " AND "))
		}
		if len(flags.Search) > 2 {
			searchConditions = append(searchConditions, strings.Join(commentConditions, " AND "))
		}

		jqlFields = append(jqlFields, fmt.Sprintf("((%s))", strings.Join(searchConditions, ") OR (")))
	}

	// Filter by assignee if specified
	if flags.Username != "" {
		jqlFields = append(jqlFields, fmt.Sprintf("assignee = '%s'", flags.Username))
	}

	// Only include issues with open sprints if the flag is set
	if flags.Sprint {
		jqlFields = append(jqlFields, "Sprint in openSprints()")
	}

	// Filter unresolved issues if the flag is set
	if flags.Unresolved {
		jqlFields = append(jqlFields, "statusCategory != 3")
	}

	// Filter by status if specified
	if flags.Status != "" {
		jqlFields = append(jqlFields, fmt.Sprintf("status = '%s'", flags.Status))
	}

	// Default filter if no filters are provided
	if len(jqlFields) == 0 {
		jqlFields = append(jqlFields, "assignee = currentUser()")
		jqlFields = append(jqlFields, "statusCategory != 3")
	}

	// Determine the ORDER BY clause based on the flag count
	orderBy := "ORDER BY key ASC" // Default ordering by key

	// If the -T flag was specified once, sort by updated DESC
	// If the -T flag was specified twice, sort by updated ASC (reverse order)
	if len(flags.OrderByTime) == 1 {
		orderBy = "ORDER BY updated DESC"
	} else if len(flags.OrderByTime) == 2 {
		orderBy = "ORDER BY updated ASC"
	}

	// Return the full JQL query with the ORDER BY clause
	return fmt.Sprintf("%s %s", strings.Join(jqlFields, " AND "), orderBy)
}
