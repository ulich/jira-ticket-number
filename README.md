# jira-ticket-number

A small command line application that reads a Jira subtask via the API and returns the Ticket number in the form of `<parent ticket number>/<subtask ticket number>`.

## Configuration

Create a configuration file at `~/.jira-ticket-number.json`:

```json
{
  "JIRA_URL": "https://jira.example.com",
  "USERNAME": "your-email@example.com",
  "PERSONAL_TOKEN": "your-api-token",
  "projectKey": "ABC"
}
```

The `projectKey` field is optional. If configured, it allows you to use just the ticket number without the project prefix (e.g., `2354` instead of `ABC-2354`).

## Usage

You can provide a ticket key, a numeric ticket number (if projectKey is configured), or a full Jira URL:

### Using a full ticket key

    jira-ticket-number ABC-2354

Will output

    ABC-1234/ABC-2354

### Using just the ticket number (requires projectKey in config)

    jira-ticket-number 2354

Will output

    ABC-1234/ABC-2354

### Using a Jira URL

    jira-ticket-number https://jira.example.com/browse/ABC-2354

Will output

    ABC-1234/ABC-2354

## Development

Run tests

    go test

Compile and run

    go run main.go ABC-2354

Build binary

    mkdir -p bin
    go build -o bin/jira-ticket-number
