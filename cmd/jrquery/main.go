package main

import (
	"fmt"
	"log"
	"os/exec"

	"irontec.com/jrquery/config"
	"irontec.com/jrquery/internal/jira"
)

var Version = "development"
var Commit = "unknown"

func main() {
	//Parse command-line flags
	flags, searchTerms, err := config.ParseFlags()
	if err != nil {
		log.Fatalf("error parsing flags: %v", err)
	}

	// If the --version flag is set, print the version and exit
	if flags.Version {
		fmt.Printf("\033[1;36mjrquery\033[0m (%s) - Jira Issues query tool [%s]\n\n", Version, Commit)

		fmt.Println("Copyright (C) 2024 Irontec S.L.")
		fmt.Println("Licenced under GPLv3+: GNU GPL version 3 or greater.")
		fmt.Println("Coded by ChatGPT <https://chatgpt.com/>")
		fmt.Println("Prompted by Ivan Alonso <kaian@irontec.com>")
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		// Check if all config values are present; if not, prompt user and save
		if err := config.PromptConfig(); err != nil {
			log.Fatalf("error saving config: %v", err)
		}

		// Try again loading the config
		cfg, err = config.LoadConfig()
		if err != nil {
			log.Fatalf("error obtaining config: %v", err)
		}
	}

	if flags.Open != "" {
		cmd := exec.Command("xdg-open", fmt.Sprintf("%s/browse/%s", cfg.JiraBaseURL, flags.Open))
		cmd.Run()
		return
	}

	// Handle 'me' as username
	if flags.Username == "me" {
		flags.Username = cfg.JiraUserEmail
	}

	// Initialize Jira client with loaded config
	client, err := jira.NewClient(cfg.JiraBaseURL, cfg.JiraAPIToken, cfg.JiraUserEmail)
	if err != nil {
		log.Fatalf("error initializing Jira client: %v", err)
	}

	// List projects
	if flags.ListProjects {
		projects, err := client.GetAllProjects()
		if err != nil {
			log.Fatalf("Error fetching projects: %v", err)
		}
		projects.Print()
		return
	}

	if flags.ListUsers {
		// Fetch the list of users
		users, err := client.GetAllUsers()
		if err != nil {
			log.Fatalf("Error fetching users: %v", err)
		}
		users.Print()
		return
	}

	// Handle the --list-filters flag to list all Jira filters
	if flags.ListFilters {
		filters, err := client.GetAllFilters()
		if err != nil {
			log.Fatalf("Error fetching filters: %v", err)
		}
		filters.Print()
		return
	}

	// Handle the --print-filter flag
	if flags.PrintFilter != 0 {
		filter, err := client.GetFilter(flags.PrintFilter)
		if err != nil {
			log.Fatalf("Error retrieving filter: %v", err)
		}
		fmt.Printf("%s\n", filter.Jql)
		return
	}

	// Build JQL query from flags
	builder := jira.NewQueryBuilder()
	jqlQuery := builder.BuildJQLQuery(flags, searchTerms)
	if flags.Debug {
		fmt.Printf("Searching issues for JQL: %s\n", jqlQuery)
	}

	var issueList *jira.IssueList
	if flags.Filter != "" {
		// Perform search using a saved filter
		issueList, err = client.SearchIssuesByFilter(flags.Filter, flags.Limit)
	} else {
		// Get tickets using the constructed JQL query
		issueList, err = client.SearchIssuesWithPagination(jqlQuery, flags.Limit)
	}

	if err != nil {
		log.Fatalf("error fetching issues: %v", err)
	}

	if flags.Count {
		// Use the Count method to get the number of issues
		fmt.Println(issueList.Count())
		return
	}

	// Print the issues to the console
	issueList.Print()
}
