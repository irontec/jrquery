package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sort"

	"irontec.com/jquery/config"
	"irontec.com/jquery/internal/jira"
)

func main() {
	//Parse command-line flags
	flags, searchTerms, err := config.ParseFlags()
	if err != nil {
		log.Fatalf("error parsing flags: %v", err)
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

	if flags.Limit > 100 {
		fmt.Println("Searchs above 100 results are not currently supported")
		return
	}

	// Initialize Jira client with loaded config
	client, err := jira.NewClient(cfg.JiraBaseURL, cfg.JiraAPIToken, cfg.JiraUserEmail)
	if err != nil {
		log.Fatalf("error initializing Jira client: %v", err)
	}

	// List projects
	if flags.ListProjects {
		projects, err := client.GetAllProjects(context.Background())
		if err != nil {
			fmt.Println("Error fetching projects:", err)
			return
		}

		// Sort the projects by their Key
		sort.Slice(projects, func(i, j int) bool {
			return (projects)[i].Key < (projects)[j].Key
		})

		// Display sorted projects with their key and name
		for _, project := range projects {
			fmt.Printf("\033[34m%-10s\033[0m: \033[33m%s\033[0m\n", project.Key, project.Name)
		}
		return
	}

	// Build JQL query from flags
	builder := jira.NewQueryBuilder()
	jqlQuery := builder.BuildJQLQuery(flags, searchTerms)
	if flags.Debug {
		fmt.Printf("Searching issues for JQL: %s\n", jqlQuery)
	}

	// Get tickets using the constructed JQL query
	issueList, err := client.SearchIssues(context.Background(), jqlQuery, 0, flags.Limit)
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
