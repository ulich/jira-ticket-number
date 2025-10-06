package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type JiraClient struct {
	BaseURL string
	Email   string
	Token   string
	Client  *http.Client
}

type JiraIssueFields struct {
	Parent *JiraParent `json:"parent,omitempty"`
}

type JiraParent struct {
	Key string `json:"key"`
}

type JiraIssue struct {
	Key    string          `json:"key"`
	Fields JiraIssueFields `json:"fields"`
}

func NewJiraClient(baseURL, email, token string, client *http.Client) *JiraClient {
	if client == nil {
		client = &http.Client{}
	}
	return &JiraClient{
		BaseURL: baseURL,
		Email:   email,
		Token:   token,
		Client:  client,
	}
}

func (c *JiraClient) GetIssue(issueKey string) (*JiraIssue, error) {
	url := fmt.Sprintf("%s/rest/api/2/issue/%s?fields=parent", c.BaseURL, issueKey)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var issue JiraIssue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &issue, nil
}

func (c *JiraClient) GetTicketOrParent(issueKey string) (string, error) {
	issue, err := c.GetIssue(issueKey)
	if err != nil {
		return "", err
	}

	if issue.Fields.Parent != nil {
		return fmt.Sprintf("%s/%s", issue.Fields.Parent.Key, issueKey), nil
	}

	return issueKey, nil
}

type Config struct {
	JIRA_URL       string
	USERNAME       string
	PERSONAL_TOKEN string
}

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	if config.JIRA_URL == "" || config.USERNAME == "" || config.PERSONAL_TOKEN == "" {
		return nil, fmt.Errorf("missing required fields in config file")
	}

	return &config, nil
}

func extractTicketKey(input string) (string, error) {
	// Try to parse as URL
	parsedURL, err := url.Parse(input)
	if err == nil && parsedURL.Scheme != "" && parsedURL.Host != "" {
		// It's a URL, extract ticket key from path
		// Expected format: /browse/TICKET-KEY
		parts := strings.Split(parsedURL.Path, "/")
		for i, part := range parts {
			if part == "browse" && i+1 < len(parts) {
				ticketKey := parts[i+1]
				if ticketKey == "" {
					return "", fmt.Errorf("could not extract ticket key from URL: %s", input)
				}
				return ticketKey, nil
			}
		}
		return "", fmt.Errorf("could not extract ticket key from URL: %s", input)
	}

	// Not a URL, treat as ticket key
	return input, nil
}

func start() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: jira-ticket-number <TICKET-KEY or URL>")
	}

	input := os.Args[1]
	ticketKey, err := extractTicketKey(input)
	if err != nil {
		return err
	}

	ticketKey = strings.ToUpper(ticketKey)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %v\n", err)
	}

	configPath := homeDir + "/.jira-ticket-number.json"
	config, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("loading %s: %v\n", configPath, err)
	}
	
	client := NewJiraClient(config.JIRA_URL, config.USERNAME, config.PERSONAL_TOKEN, nil)

	result, err := client.GetTicketOrParent(ticketKey)
	if err != nil {
		return fmt.Errorf("get ticket: %v\n", err)
	}

	fmt.Println(result)
	return nil
}

func main() {
	err := start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
