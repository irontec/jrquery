package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	jira "github.com/andygrunwald/go-jira/v2/cloud"
	flags "github.com/jessevdk/go-flags"
	"github.com/spf13/viper"
)

type Options struct {
	Debug      bool   `short:"d" long:"debug" description:"Print debugging information"`
	Username   string `short:"u" long:"user" description:"Name or email of assigned user"`
	Project    string `short:"p" long:"project" description:"Key of project to search issues"`
	Search     []bool `short:"s" long:"search" description:"Search text in summary, issue description or comments"`
	Limit      int    `short:"l" long:"limit" default:"50" description:"Limit output to first N results"`
	Count      bool   `short:"c" long:"count" description:"Only print issue count"`
	Sprint     bool   `short:"S" long:"sprint" description:"Only print issues with active sprint"`
	Status     string `short:"e" long:"status" description:"Only print issues with given status Name"`
	Unresolved bool   `short:"O" long:"unresolved" description:"Only print unresolved issues"`
	All        bool   `short:"A" long:"all" description:"Print all issues no matter their status"`
	Query      string `short:"q" long:"query" description:"Run a custom query"`
	Open       string `short:"o" long:"open" description:"Open given issue in a browser tab"`
}

type DisplayOptions struct {
	KeyWidth    int
	StatusWidth int
}

var options Options

var parser = flags.NewParser(&options, flags.Default)

// Function to get the user's configuration file path
func getUserConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get the user's home directory: %v", err)
	}
	return filepath.Join(homeDir, ".config", "jquery.json"), nil
}

// Function to prompt the user for configuration values
func promptForConfig() (string, string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("You need a Jira API token to use jquery.")
	fmt.Println()
	fmt.Println("\033[1;37mhttps://id.atlassian.com/manage-profile/security/api-tokens\033[0m")
	fmt.Println()

	// Prompt for Jira BaseURL
	fmt.Print("Enter Jira BaseURL: ")
	baseURL, _ := reader.ReadString('\n')
	baseURL = strings.TrimSpace(baseURL)

	// Prompt for the Username
	fmt.Print("Enter your email: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	// Prompt for the APIToken
	fmt.Print("Enter your APIToken: ")
	apiToken, _ := reader.ReadString('\n')
	apiToken = strings.TrimSpace(apiToken)

	return baseURL, username, apiToken
}

// Function to save the configuration to a file
func saveConfig(baseURL, username, apiToken, configPath string) error {
	viper.Set("BaseURL", baseURL)
	viper.Set("Username", username)
	viper.Set("APIToken", apiToken)

	// Ensure the directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		return fmt.Errorf("could not create configuration directory: %v", err)
	}

	// Write the configuration to the file
	viper.SetConfigFile(configPath)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("could not write the configuration file: %v", err)
	}

	return nil
}

func buildJQL(searchTerms []string, options Options) string {

	var jqlFields []string

	if options.Query != "" {
		return options.Query
	}

	if options.Project != "" {
		jqlFields = append(jqlFields, fmt.Sprintf("project = '%s'", options.Project))
	}

	if len(options.Search) > 0 {
		var summaryConditions []string
		var descriptionConditions []string
		var commentConditions []string

		for _, term := range searchTerms {
			summaryConditions = append(summaryConditions, fmt.Sprintf("summary ~ '%s'", term))
			descriptionConditions = append(descriptionConditions, fmt.Sprintf("description ~ '%s'", term))
			commentConditions = append(commentConditions, fmt.Sprintf("comment ~ '%s'", term))
		}

		var searchConditions []string
		if len(options.Search) > 0 {
			searchConditions = append(searchConditions, strings.Join(summaryConditions, " AND "))
		}
		if len(options.Search) > 1 {
			searchConditions = append(searchConditions, strings.Join(descriptionConditions, " AND "))
		}
		if len(options.Search) > 2 {
			searchConditions = append(searchConditions, strings.Join(commentConditions, " AND "))
		}

		jqlFields = append(jqlFields, fmt.Sprintf("(%s)", strings.Join(searchConditions, ") OR (")))
	}

	if options.Username != "" {
		jqlFields = append(jqlFields, fmt.Sprintf("assignee = '%s'", options.Username))
	}

	if options.Sprint {
		jqlFields = append(jqlFields, "Sprint in openSprints()")
	}

	if options.Unresolved {
		jqlFields = append(jqlFields, "statusCategory != 3")
	}

	if options.Status != "" {
		jqlFields = append(jqlFields, fmt.Sprintf("status = '%s'", options.Status))
	}

	if len(jqlFields) == 0 {
		jqlFields = append(jqlFields, "assignee = currentUser()")
		jqlFields = append(jqlFields, "statusCategory != 3")
	}

	return fmt.Sprintf("%s ORDER BY key ASC", strings.Join(jqlFields, " AND "))
}

func printIssue(issue jira.Issue, options DisplayOptions) {
	issueKeyColor := "\033[1;34m"

	if issue.Fields.Status.StatusCategory.Key == "new" {
		issueKeyColor = "\033[1;37m"
	}

	if issue.Fields.Status.StatusCategory.Key == "done" {
		issueKeyColor = "\033[1;32m"
	}

	assigneeName := ""
	if issue.Fields.Assignee != nil {
		assigneeName = issue.Fields.Assignee.DisplayName
	}

	fmt.Printf(
		"[%s%-*s\033[0m][%s%-*s\033[0m][%s][\033[34m%s\033[0m](\033[33m%s\033[0m)\033[1;37m %s\033[0m\n",
		issueKeyColor,
		options.KeyWidth,
		issue.Key,
		issueKeyColor,
		options.StatusWidth,
		issue.Fields.Status.Name,
		time.Time(issue.Fields.Created).Format("02-01-2006"),
		assigneeName,
		issue.Fields.Project.Name,
		issue.Fields.Summary,
	)
}

func main() {
	searchTerms, err := parser.Parse()
	if err != nil {
		switch flagsErr := err.(type) {
		case flags.ErrorType:
			if flagsErr == flags.ErrHelp {
				os.Exit(0)
			}
			os.Exit(1)
		default:
			os.Exit(1)
		}
	}

	// Get the configuration file path
	configPath, err := getUserConfigPath()
	if err != nil {
		fmt.Println("Error getting configuration file path:", err)
		return
	}

	// Configure Viper
	viper.SetConfigFile(configPath)
	viper.SetConfigType("json")

	// Try to read the configuration file
	if err := viper.ReadInConfig(); err != nil {
		// If the file doesn't exist or there's an error, prompt the user
		fmt.Println("Configuration file not found or error reading it.")
		baseURL, username, apiToken := promptForConfig()

		// Validate entered data
		tp := jira.BasicAuthTransport{
			Username: username,
			APIToken: apiToken,
		}

		client, err := jira.NewClient(baseURL, tp.Client())
		if err != nil {
			fmt.Println("Unable to validate entered information:", err)
			return
		}

		currentUser, _, err := client.User.GetCurrentUser(context.Background())
		if err != nil {
			fmt.Println("Unable to get current user information:", err)
			return
		}

		fmt.Println("Confguration validated for user: ", currentUser.DisplayName)

		// Save the configuration to the file
		if err := saveConfig(baseURL, username, apiToken, configPath); err != nil {
			fmt.Println("Error saving the configuration:", err)
			return
		}

		fmt.Println("Configuration successfully saved to", configPath)
	}

	if options.Open != "" {
		cmd := exec.Command("xdg-open", fmt.Sprintf("%s/browse/%s", viper.GetString("BaseURL"), options.Open))
		cmd.Run()
		return
	}

	if options.Limit > 100 {
		fmt.Println("Searchs above 100 results are not currently supported")
		return
	}

	tp := jira.BasicAuthTransport{
		Username: viper.GetString("Username"),
		APIToken: viper.GetString("APIToken"),
	}

	client, err := jira.NewClient(viper.GetString("BaseURL"), tp.Client())
	if err != nil {
		panic(err)
	}

	jql := buildJQL(searchTerms, options)
	if options.Debug {
		fmt.Printf("Searching issues for JQL: %s\n", jql)
	}

	searchOptions := &jira.SearchOptions{
		MaxResults: options.Limit,
	}

	issues, response, err := client.Issue.Search(context.Background(), jql, searchOptions)
	if err != nil {
		log.Fatalf("Error searching tickets: %v", err)
	}

	if options.Count {
		fmt.Println(len(issues))
		return
	}

	if len(issues) == 0 {
		fmt.Println("No results found.")
		return
	}

	displayOptions := DisplayOptions{
		KeyWidth:    0,
		StatusWidth: 0,
	}

	for _, issue := range issues {
		if len(issue.Key) > displayOptions.KeyWidth {
			displayOptions.KeyWidth = len(issue.Key)
		}
		if len(issue.Fields.Status.Name) > displayOptions.StatusWidth {
			displayOptions.StatusWidth = len(issue.Fields.Status.Name)
		}
	}

	for _, issue := range issues {
		printIssue(issue, displayOptions)
	}

	if response.Total > response.MaxResults {
		fmt.Printf("Displaying first %d results of %d\n", response.MaxResults, response.Total)
	}
}
