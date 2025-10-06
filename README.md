# jira-ticket-number

A small command line application that reads a Jira subtask via the API and returns the Ticket number in the form of `<parent ticket number>/<subtask ticket number>`.

## Usage

You can provide either a ticket key or a full Jira URL:

### Using a ticket key

    jira-ticket-number ABC-2354

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

    go build -o bin/jira-ticket-number
