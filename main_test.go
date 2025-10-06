package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type mockRoundTripper struct {
	responses map[string]*http.Response
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, ok := m.responses[req.URL.String()]
	if !ok {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       http.NoBody,
		}, nil
	}
	return resp, nil
}

func newMockResponse(statusCode int, body string) *http.Response {
	resp := httptest.NewRecorder()
	resp.WriteHeader(statusCode)
	resp.WriteString(body)
	return resp.Result()
}

func TestGetIssue_WithParent(t *testing.T) {
	mockResp := `{
		"key": "ABC-2732",
		"fields": {
			"parent": {
				"key": "ABC-2181"
			}
		}
	}`

	transport := &mockRoundTripper{
		responses: map[string]*http.Response{
			"https://jira.example.com/rest/api/2/issue/ABC-2732?fields=parent": newMockResponse(http.StatusOK, mockResp),
		},
	}

	client := NewJiraClient("https://jira.example.com", "token", &http.Client{
		Transport: transport,
	})

	issue, err := client.GetIssue("ABC-2732")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if issue.Key != "ABC-2732" {
		t.Errorf("expected issue key ABC-2732, got: %s", issue.Key)
	}

	if issue.Fields.Parent == nil {
		t.Fatal("expected parent to be present")
	}

	if issue.Fields.Parent.Key != "ABC-2181" {
		t.Errorf("expected parent key ABC-2181, got: %s", issue.Fields.Parent.Key)
	}
}

func TestGetIssue_WithoutParent(t *testing.T) {
	mockResp := `{
		"key": "ABC-2181",
		"fields": {}
	}`

	transport := &mockRoundTripper{
		responses: map[string]*http.Response{
			"https://jira.example.com/rest/api/2/issue/ABC-2181?fields=parent": newMockResponse(http.StatusOK, mockResp),
		},
	}

	client := NewJiraClient("https://jira.example.com", "token", &http.Client{
		Transport: transport,
	})

	issue, err := client.GetIssue("ABC-2181")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if issue.Key != "ABC-2181" {
		t.Errorf("expected issue key ABC-2181, got: %s", issue.Key)
	}

	if issue.Fields.Parent != nil {
		t.Errorf("expected no parent, got: %s", issue.Fields.Parent.Key)
	}
}

func TestGetTicketOrParent_Subtask(t *testing.T) {
	mockResp := `{
		"key": "ABC-2732",
		"fields": {
			"parent": {
				"key": "ABC-2181"
			}
		}
	}`

	transport := &mockRoundTripper{
		responses: map[string]*http.Response{
			"https://jira.example.com/rest/api/2/issue/ABC-2732?fields=parent": newMockResponse(http.StatusOK, mockResp),
		},
	}

	client := NewJiraClient("https://jira.example.com", "token", &http.Client{
		Transport: transport,
	})

	result, err := client.GetTicketOrParent("ABC-2732")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "ABC-2181/ABC-2732"
	if result != expected {
		t.Errorf("expected %s, got: %s", expected, result)
	}
}

func TestGetTicketOrParent_ParentTask(t *testing.T) {
	mockResp := `{
		"key": "ABC-2181",
		"fields": {}
	}`

	transport := &mockRoundTripper{
		responses: map[string]*http.Response{
			"https://jira.example.com/rest/api/2/issue/ABC-2181?fields=parent": newMockResponse(http.StatusOK, mockResp),
		},
	}

	client := NewJiraClient("https://jira.example.com", "token", &http.Client{
		Transport: transport,
	})

	result, err := client.GetTicketOrParent("ABC-2181")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "ABC-2181"
	if result != expected {
		t.Errorf("expected %s, got: %s", expected, result)
	}
}

func TestGetIssue_APIError(t *testing.T) {
	transport := &mockRoundTripper{
		responses: map[string]*http.Response{
			"https://jira.example.com/rest/api/2/issue/INVALID?fields=parent": newMockResponse(http.StatusNotFound, `{"errorMessages":["Issue does not exist"]}`),
		},
	}

	client := NewJiraClient("https://jira.example.com", "token", &http.Client{
		Transport: transport,
	})

	_, err := client.GetIssue("INVALID")
	if err == nil {
		t.Fatal("expected error for invalid issue, got nil")
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	content := `{
	"jiraUrl": "https://jira.example.com",
	"personalToken": "token123"
}`
	tmpfile := "/tmp/test_config.json"
	if err := writeTestFile(tmpfile, content); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config, err := loadConfig(tmpfile)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if config.JiraURL != "https://jira.example.com" {
		t.Errorf("expected JiraURL to be https://jira.example.com, got: %s", config.JiraURL)
	}

	if config.PersonalToken != "token123" {
		t.Errorf("expected PersonalToken to be token123, got: %s", config.PersonalToken)
	}
}

func writeTestFile(path, content string) error {
	f, err := openFile(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

func openFile(path string) (*os.File, error) {
	return os.Create(path)
}

func TestExtractTicketKey_FromTicketKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"uppercase ticket", "ABC-2247", "ABC-2247"},
		{"lowercase ticket", "ABC-2247", "ABC-2247"},
		{"mixed case ticket", "ABC-2247", "ABC-2247"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractTicketKey(tt.input, "")
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got: %s", tt.expected, result)
			}
		})
	}
}

func TestExtractTicketKey_FromURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"standard Jira URL",
			"https://jira.example.com/browse/ABC-2247",
			"ABC-2247",
		},
		{
			"URL with query params",
			"https://jira.example.com/browse/ABC-2247?filter=all",
			"ABC-2247",
		},
		{
			"URL with fragment",
			"https://jira.example.com/browse/ABC-2247#comment-12345",
			"ABC-2247",
		},
		{
			"different domain",
			"https://jira.example.com/browse/ABC-123",
			"ABC-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractTicketKey(tt.input, "")
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got: %s", tt.expected, result)
			}
		})
	}
}

func TestExtractTicketKey_InvalidURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"URL without browse path", "https://jira.example.com/issues/ABC-2247"},
		{"URL with incomplete path", "https://jira.example.com/browse/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := extractTicketKey(tt.input, "")
			if err == nil {
				t.Fatal("expected error for invalid URL format, got nil")
			}
		})
	}
}

func TestExtractTicketKey_NumericOnly(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		projectKey string
		expected   string
		wantErr    bool
	}{
		{
			"with project key",
			"2803",
			"ABC",
			"ABC-2803",
			false,
		},
		{
			"with different project",
			"DEF-2803",
			"ABC",
			"DEF-2803",
			false,
		},
		{
			"numeric without project key",
			"2803",
			"",
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractTicketKey(tt.input, tt.projectKey)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got: %s", tt.expected, result)
			}
		})
	}
}
