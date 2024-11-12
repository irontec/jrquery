package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config struct holds the application configuration
type Config struct {
	JiraBaseURL   string
	JiraAPIToken  string
	JiraUserEmail string
}

func getUserConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get the user's home directory: %v", err)
	}
	return filepath.Join(homeDir, ".config", "jquery.json"), nil
}

// LoadConfig loads configuration from environment variables or a config file.
func LoadConfig() (*Config, error) {

	// Get the configuration file path
	configPath, err := getUserConfigPath()
	if err != nil {
		fmt.Println("Error getting configuration file path:", err)
		return nil, err
	}

	// Configure Viper
	viper.SetConfigFile(configPath)
	viper.SetConfigType("json")

	// Set default values
	viper.SetDefault("jira.base_url", "")
	viper.SetDefault("jira.api_token", "")
	viper.SetDefault("jira.user_email", "")

	// Enable reading from environment variables
	viper.AutomaticEnv()

	// Map environment variables to viper keys
	viper.BindEnv("jira.base_url", "JIRA_BASE_URL")
	viper.BindEnv("jira.api_token", "JIRA_API_TOKEN")
	viper.BindEnv("jira.user_email", "JIRA_USER_EMAIL")

	// Attempt to read from config file, if exists
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	// Return the configuration
	config := &Config{
		JiraBaseURL:   viper.GetString("jira.base_url"),
		JiraAPIToken:  viper.GetString("jira.api_token"),
		JiraUserEmail: viper.GetString("jira.user_email"),
	}

	return config, nil
}

// SaveConfig saves the current configuration to a file.
func SaveConfig(cfg *Config) error {
	viper.Set("jira.base_url", cfg.JiraBaseURL)
	viper.Set("jira.api_token", cfg.JiraAPIToken)
	viper.Set("jira.user_email", cfg.JiraUserEmail)

	// Get the configuration file path
	configPath, err := getUserConfigPath()
	if err != nil {
		fmt.Println("Error getting configuration file path:", err)
		return err
	}

	// Ensure the directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		return fmt.Errorf("could not create configuration directory: %v", err)
	}

	// Write the configuration to the file
	return viper.WriteConfig()
}

// CheckAndPromptConfig checks if configuration is present; if not, prompts user for input and saves it.
func PromptConfig() error {

	config := &Config{}
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("You need a Jira API token to use jquery.")
	fmt.Println()
	fmt.Println("\033[1;37mhttps://id.atlassian.com/manage-profile/security/api-tokens\033[0m")
	fmt.Println()

	// Prompt for Jira BaseURL
	if config.JiraBaseURL == "" {
		fmt.Print("Enter Jira BaseURL: ")
		url, _ := reader.ReadString('\n')
		config.JiraBaseURL = strings.TrimSpace(url)
	}

	// Prompt for the Username
	if config.JiraUserEmail == "" {
		fmt.Print("Enter your email: ")
		email, _ := reader.ReadString('\n')
		config.JiraUserEmail = strings.TrimSpace(email)
	}

	// Prompt for the APIToken
	if config.JiraAPIToken == "" {
		fmt.Print("Enter your APIToken: ")
		token, _ := reader.ReadString('\n')
		config.JiraAPIToken = strings.TrimSpace(token)
	}

	// Save the new config if any of the values were prompted
	return SaveConfig(config)
}
