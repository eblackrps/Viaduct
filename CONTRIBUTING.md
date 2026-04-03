# Contributing

## How to Report Bugs
Open a GitHub issue with a clear summary, the environment you were using, reproduction steps, expected behavior, and the actual behavior you observed. If logs or screenshots help clarify the problem, include them with any secrets removed.

## How to Request Features
Open a GitHub issue describing the problem you are trying to solve, why existing behavior is not enough, and what a successful outcome would look like. Feature requests with operational context are easier to prioritize than solution-only requests.

## Development Setup
- Go 1.22 or newer
- `make`
- `golangci-lint`

Clone the repository, install dependencies with `go mod tidy`, and use the Make targets for the standard workflow.

If you use Codex for repo tasks, the bootstrap script in `.codex/setup.sh` installs the expected linter version in supported environments.

## Workflow
1. Fork the repository.
2. Create a feature branch for your work.
3. Make your changes with focused commits.
4. Run `make all`.
5. Submit a pull request with context about the problem, solution, and verification.

## Security
If you believe you have found a security issue, follow the process in [SECURITY.md](SECURITY.md) instead of opening a public bug report.
