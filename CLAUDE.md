# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build/Test/Lint Commands
- Build all components: `cd engine && make build`
- Lint code: `cd engine && make lint`
- Run unit tests: `cd engine && make test`
- Run integration tests: `cd engine && make test-ci-integration`
- Run a specific test: `cd engine && GO111MODULE=on go test -v ./path/to/package -run TestName`
- Run UI: `cd ui && pnpm start:ce` (Community Edition) or `pnpm start:platform`

## Code Style Guidelines
- Go code follows "Effective Go" and "Go Code Review Comments" guidelines
- Use present tense and imperative mood in commit messages
- Limit first commit line to 72 characters
- All Git commits must be signed
- Format Go code with `cd engine && make fmt`
- Use error handling with pkg/errors
- Follow standard Go import ordering
- Group similar functions together
- Error messages should be descriptive and actionable
- UI uses pnpm for package management