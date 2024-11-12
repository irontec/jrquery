package jira

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/andygrunwald/go-jira/v2/cloud"
)

// UserList holds a list of Jira users and provides methods for displaying them.
type UserList struct {
	Users      []cloud.User
	MaxResults int
	Total      int
}

// NewUserList initializes a new UserList with a given slice of users.
func NewUserList(users []cloud.User, max, total int) *UserList {
	return &UserList{Users: users, MaxResults: max, Total: total}
}

// Count returns the number of users in the list.
func (ul *UserList) Count() int {
	return len(ul.Users)
}

// Print displays the users on the console.
func (ul *UserList) Print() {
	if len(ul.Users) == 0 {
		fmt.Println("No users found.")
		return
	}

	// Sort the users by their DisplayName
	sort.Slice(ul.Users, func(i, j int) bool {
		return ul.Users[i].DisplayName < ul.Users[j].DisplayName
	})

	for _, user := range ul.Users {
		if user.AccountType == "atlassian" && user.Active {
			fmt.Printf("\033[1;34m%s\033[0m: \033[33m%s\033[0m\n", user.EmailAddress, user.DisplayName)
		}
	}

	if ul.Total > ul.MaxResults {
		fmt.Printf("\033[1;32m * \033[1;31mDisplaying first %d of %d users\033[0m\n", ul.MaxResults, ul.Total)
	}
}

// ToJSON converts the UserList to a JSON representation.
func (ul *UserList) ToJSON() (string, error) {
	data, err := json.MarshalIndent(ul, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error converting users to JSON: %w", err)
	}
	return string(data), nil
}
