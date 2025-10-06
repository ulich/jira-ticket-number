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
		"key": "PNC-2732",
		"fields": {
			"parent": {
				"key": "PNC-2181"
			}
		}
	}`

	transport := &mockRoundTripper{
		responses: map[string]*http.Response{
			"https://jira.example.com/rest/api/2/issue/PNC-2732?fields=parent": newMockResponse(http.StatusOK, mockResp),
		},
	}

	client := NewJiraClient("https://jira.example.com", "test@example.com", "token", &http.Client{
		Transport: transport,
	})

	issue, err := client.GetIssue("PNC-2732")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if issue.Key != "PNC-2732" {
		t.Errorf("expected issue key PNC-2732, got: %s", issue.Key)
	}

	if issue.Fields.Parent == nil {
		t.Fatal("expected parent to be present")
	}

	if issue.Fields.Parent.Key != "PNC-2181" {
		t.Errorf("expected parent key PNC-2181, got: %s", issue.Fields.Parent.Key)
	}
}

func TestGetIssue_WithoutParent(t *testing.T) {
	mockResp := `{
		"key": "PNC-2181",
		"fields": {}
	}`

	transport := &mockRoundTripper{
		responses: map[string]*http.Response{
			"https://jira.example.com/rest/api/2/issue/PNC-2181?fields=parent": newMockResponse(http.StatusOK, mockResp),
		},
	}

	client := NewJiraClient("https://jira.example.com", "test@example.com", "token", &http.Client{
		Transport: transport,
	})

	issue, err := client.GetIssue("PNC-2181")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if issue.Key != "PNC-2181" {
		t.Errorf("expected issue key PNC-2181, got: %s", issue.Key)
	}

	if issue.Fields.Parent != nil {
		t.Errorf("expected no parent, got: %s", issue.Fields.Parent.Key)
	}
}

func TestGetTicketOrParent_Subtask(t *testing.T) {
	mockResp := `{
		"key": "PNC-2732",
		"fields": {
			"parent": {
				"key": "PNC-2181"
			}
		}
	}`

	transport := &mockRoundTripper{
		responses: map[string]*http.Response{
			"https://jira.example.com/rest/api/2/issue/PNC-2732?fields=parent": newMockResponse(http.StatusOK, mockResp),
		},
	}

	client := NewJiraClient("https://jira.example.com", "test@example.com", "token", &http.Client{
		Transport: transport,
	})

	result, err := client.GetTicketOrParent("PNC-2732")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "PNC-2181/PNC-2732"
	if result != expected {
		t.Errorf("expected %s, got: %s", expected, result)
	}
}

func TestGetTicketOrParent_ParentTask(t *testing.T) {
	mockResp := `{
		"key": "PNC-2181",
		"fields": {}
	}`

	transport := &mockRoundTripper{
		responses: map[string]*http.Response{
			"https://jira.example.com/rest/api/2/issue/PNC-2181?fields=parent": newMockResponse(http.StatusOK, mockResp),
		},
	}

	client := NewJiraClient("https://jira.example.com", "test@example.com", "token", &http.Client{
		Transport: transport,
	})

	result, err := client.GetTicketOrParent("PNC-2181")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "PNC-2181"
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

	client := NewJiraClient("https://jira.example.com", "test@example.com", "token", &http.Client{
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
	"JIRA_URL": "https://jira.example.com",
	"USERNAME": "test@example.com",
	"PERSONAL_TOKEN": "token123"
}`
	tmpfile := "/tmp/test_config.json"
	if err := writeTestFile(tmpfile, content); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config, err := loadConfig(tmpfile)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if config.JIRA_URL != "https://jira.example.com" {
		t.Errorf("expected JIRA_URL to be https://jira.example.com, got: %s", config.JIRA_URL)
	}

	if config.USERNAME != "test@example.com" {
		t.Errorf("expected USERNAME to be test@example.com, got: %s", config.USERNAME)
	}

	if config.PERSONAL_TOKEN != "token123" {
		t.Errorf("expected PERSONAL_TOKEN to be token123, got: %s", config.PERSONAL_TOKEN)
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
