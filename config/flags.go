package config

import (
	"os"

	"github.com/jessevdk/go-flags"
)

// Flags struct holds the command-line flags for the application
type Flags struct {
	Debug        bool   `short:"d" long:"debug" description:"Print debugging information"`
	Username     string `short:"u" long:"user" description:"Name or email of assigned user"`
	Project      string `short:"p" long:"project" description:"Key of project to search issues"`
	Search       []bool `short:"s" long:"search" description:"Search text in summary, issue description or comments"`
	Limit        int    `short:"l" long:"limit" default:"50" description:"Limit output to first N results"`
	Count        bool   `short:"c" long:"count" description:"Only print issue count"`
	Sprint       bool   `short:"S" long:"sprint" description:"Only print issues with active sprint"`
	Status       string `short:"e" long:"status" description:"Only print issues with given status Name"`
	Unresolved   bool   `short:"O" long:"unresolved" description:"Only print unresolved issues"`
	All          bool   `short:"A" long:"all" description:"Print all issues no matter their status"`
	Query        string `short:"q" long:"query" description:"Run a custom query"`
	Filter       string `short:"f" long:"filter" description:"Search issues using a saved Jira filter ID"`
	Open         string `short:"o" long:"open" description:"Open given issue in a browser tab"`
	OrderByTime  []bool `short:"T" long:"order-by-time" description:"Sort issues by last updated time (use -TT for reverse)"`
	OrderByUser  []bool `short:"U" long:"order-by-user" description:"Sort issues by assignee (use -UU for reverse ordering)"`
	ListProjects bool   `long:"list-projects" description:"List all visible projects for current user"`
	ListUsers    bool   `long:"list-users" description:"List all users in Jira"`
	ListFilters  bool   `long:"list-filters" description:"List all saved filters in Jira"`
	PrintFilter  int    `long:"print-filter" description:"Print the JQL query of a Jira filter by ID"`
	Version      bool   `short:"v" long:"version" description:"Show the version"`
}

// ParseFlags parses command-line flags and returns a populated Flags struct
func ParseFlags() (*Flags, []string, error) {
	var opts Flags
	var searchTerms []string
	parser := flags.NewParser(&opts, flags.Default)
	searchTerms, err := parser.Parse()
	if flags.WroteHelp(err) {
		os.Exit(0)
	}
	return &opts, searchTerms, err
}
