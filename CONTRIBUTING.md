# Contributing to AGIS Bot

Thank you for your interest in contributing to AGIS Bot! This document provides guidelines and information about contributing.

## License Notice

AGIS Bot is licensed under the [Business Source License 1.1 (BSL-1.1)](LICENSE). By contributing, you agree that your contributions will be licensed under the same terms.

**Important**: This is an Open Core project:
- `wethegamers/agis` (this repo) - Public entry point, charts, documentation
- `wethegamers/agis-core` - Private core with proprietary business logic

Contributions to this repository are welcome. Core functionality contributions require separate arrangements.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md) to maintain a welcoming and inclusive community.

## How to Contribute

### Reporting Bugs

Before creating a bug report:
1. Check the [existing issues](https://github.com/wethegamers/agis/issues) to avoid duplicates
2. Collect relevant information (logs, environment, steps to reproduce)

When creating a bug report, include:
- Clear, descriptive title
- Steps to reproduce the behavior
- Expected vs actual behavior
- Environment details (Go version, Kubernetes version, etc.)
- Relevant logs or error messages

### Suggesting Features

Feature suggestions are welcome! Please:
1. Check existing issues and discussions for similar requests
2. Provide clear use case and rationale
3. Consider how it fits with the project's goals

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Follow coding standards** (see below)
3. **Add tests** for new functionality
4. **Update documentation** as needed
5. **Write clear commit messages** (see commit guidelines below)
6. **Open a pull request** with a clear description

## Development Setup

### Prerequisites

- Go 1.24+
- Docker (for container builds)
- kubectl (for Kubernetes testing)
- Helm 3.x (for chart development)

### Local Development

```bash
# Clone the repository
git clone https://github.com/wethegamers/agis.git
cd agis

# Install dependencies
go mod download

# Run tests
make test

# Build locally
make build

# Run linting
make lint
```

### Running Tests

```bash
# Unit tests
make test

# With coverage
make test-coverage

# Integration tests (requires test environment)
make test-integration
```

## Coding Standards

### Go Code

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `gofmt` for formatting
- Use `golangci-lint` for linting
- Write meaningful comments for exported functions
- Keep functions focused and testable

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Examples:
```
feat(discord): add slash command for server stats
fix(database): resolve connection pool exhaustion
docs(readme): update deployment instructions
```

### Helm Charts

- Follow [Helm best practices](https://helm.sh/docs/chart_best_practices/)
- Include meaningful default values
- Document all values in `values.yaml`
- Test charts with `helm lint` and `helm template`

## Pull Request Process

1. Ensure all tests pass
2. Update relevant documentation
3. Add changelog entry if applicable
4. Request review from maintainers
5. Address review feedback
6. Squash commits if requested

### Review Criteria

Pull requests are evaluated on:
- Code quality and style
- Test coverage
- Documentation completeness
- Alignment with project goals
- Security implications

## Getting Help

- **Questions**: Open a [GitHub Discussion](https://github.com/wethegamers/agis/discussions)
- **Bugs**: Open a [GitHub Issue](https://github.com/wethegamers/agis/issues)
- **Security**: See [SECURITY.md](SECURITY.md)

## Recognition

Contributors are recognized in:
- Release notes
- CHANGELOG.md (for significant contributions)
- README.md contributors section (for ongoing contributors)

Thank you for contributing to AGIS Bot! 🎮
