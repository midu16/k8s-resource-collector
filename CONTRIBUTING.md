# Contributing to k8s-resource-collector

Thank you for your interest in contributing to k8s-resource-collector! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Review Process](#review-process)
- [Community](#community)

## Code of Conduct

### Our Pledge

We are committed to providing a welcoming and inclusive environment for everyone. We expect all contributors to:

- Be respectful and considerate in communication
- Accept constructive criticism gracefully
- Focus on what is best for the project and community
- Show empathy towards other community members

### Expected Behavior

- Use welcoming and inclusive language
- Be respectful of differing viewpoints and experiences
- Gracefully accept constructive feedback
- Focus on collaboration and constructive discussion

## Getting Started

### Types of Contributions We Welcome

- **Bug Reports**: Found a bug? Let us know!
- **Feature Requests**: Have an idea? We'd love to hear it!
- **Code Contributions**: Bug fixes, new features, improvements
- **Documentation**: Improvements to README, guides, examples
- **Testing**: Additional test cases, test improvements
- **Performance**: Optimizations and efficiency improvements

### Before You Start

1. **Check existing issues**: Search for existing issues or pull requests
2. **Create an issue**: Discuss major changes before starting work
3. **Ask questions**: If unsure, ask in an issue or discussion

## Development Setup

### Prerequisites

- **Go**: Version 1.21 or higher
- **Git**: For version control
- **Make**: For build automation
- **Docker** (optional): For container testing

### Initial Setup

1. **Fork the repository**
   ```bash
   # Click "Fork" on GitHub, then:
   git clone https://github.com/YOUR_USERNAME/k8s-resource-collector.git
   cd k8s-resource-collector
   ```

2. **Add upstream remote**
   ```bash
   git remote add upstream https://github.com/midu/k8s-resource-collector.git
   ```

3. **Set up development environment**
   ```bash
   make setup
   ```

4. **Verify setup**
   ```bash
   make build
   make test-unit
   ```

### Development Workflow

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/bug-description
   ```

2. **Make your changes**
   - Write code following our [coding standards](#coding-standards)
   - Add tests for new functionality
   - Update documentation as needed

3. **Run tests locally**
   ```bash
   make test-unit        # Run unit tests
   make lint             # Run linters
   make fmt              # Format code
   ```

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add new feature description"
   ```

## Making Changes

### Branch Naming

Use descriptive branch names with prefixes:

- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test additions or modifications
- `chore/` - Maintenance tasks

Examples:
```
feature/multi-cluster-comparison
fix/kubeconfig-validation-error
docs/update-readme-examples
```

### Commit Message Format

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `perf`: Performance improvements

**Examples:**
```
feat(comparison): add multi-cluster comparison mode

Add support for comparing resources between two Kubernetes clusters.
Includes automatic cluster name detection and diff generation.

Closes #123
```

```
fix(collection): handle timeout errors gracefully

Add proper error handling for timeout scenarios during resource collection.
Includes retry logic with exponential backoff.

Fixes #456
```

## Coding Standards

### Go Code Style

1. **Follow Go conventions**
   - Use `gofmt` for formatting
   - Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
   - Use meaningful variable and function names

2. **Code organization**
   ```go
   // Package-level documentation
   package main

   // Imports grouped: standard library, external, internal
   import (
       "fmt"
       "os"
       
       "k8s.io/client-go/kubernetes"
       
       "github.com/midu/k8s-resource-collector/internal/utils"
   )

   // Constants and variables
   const defaultTimeout = 30 * time.Second

   // Main types and functions
   ```

3. **Error handling**
   ```go
   // Good: Wrap errors with context
   if err != nil {
       return fmt.Errorf("failed to parse kubeconfig: %w", err)
   }

   // Bad: Generic errors
   if err != nil {
       return err
   }
   ```

4. **Documentation**
   ```go
   // CollectResources collects all Kubernetes resources from the cluster
   // and saves them to the specified output directory.
   //
   // Parameters:
   //   - outputDir: Directory where resources will be saved
   //
   // Returns an error if collection fails.
   func CollectResources(outputDir string) error {
       // Implementation
   }
   ```

### File Organization

```
k8s-resource-collector/
├── cmd/                    # Main application code
│   └── main.go
├── internal/              # Private application code
│   ├── collector/         # Collection logic
│   ├── formatter/         # Output formatting
│   └── utils/            # Utility functions
├── tests/                 # Test files
├── docs/                  # Additional documentation
└── examples/             # Example configurations
```

### Code Quality Requirements

1. **All code must pass linting**
   ```bash
   make lint
   ```

2. **Code must be formatted**
   ```bash
   make fmt
   ```

3. **No compiler warnings**
   ```bash
   go vet ./...
   ```

## Testing

### Test Requirements

**All contributions must include tests!**

1. **Unit Tests**: For all new functions and methods
2. **Integration Tests**: For end-to-end scenarios
3. **Test Coverage**: Maintain or improve coverage

### Writing Tests

```go
func TestCollectResources(t *testing.T) {
    tests := []struct {
        name        string
        outputDir   string
        expectedErr error
    }{
        {
            name:        "valid directory",
            outputDir:   "/tmp/test",
            expectedErr: nil,
        },
        {
            name:        "invalid directory",
            outputDir:   "/invalid/path",
            expectedErr: os.ErrPermission,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := CollectResources(tt.outputDir)
            if err != tt.expectedErr {
                t.Errorf("expected %v, got %v", tt.expectedErr, err)
            }
        })
    }
}
```

### Running Tests

```bash
# Run unit tests
make test-unit

# Run all tests
make test-all

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out
```

### Test Guidelines

1. **Use table-driven tests** when testing multiple scenarios
2. **Test edge cases** and error conditions
3. **Use descriptive test names** that explain what is being tested
4. **Clean up** any resources created during tests
5. **Mock external dependencies** (API calls, file system, etc.)

## Submitting Changes

### Pull Request Process

1. **Update your fork**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Push your changes**
   ```bash
   git push origin feature/your-feature-name
   ```

3. **Create Pull Request**
   - Go to GitHub and create a PR from your fork
   - Fill out the PR template completely
   - Link related issues using "Fixes #123" or "Closes #456"

4. **PR Checklist**
   - [ ] Code follows project style guidelines
   - [ ] All tests pass locally
   - [ ] New tests added for new functionality
   - [ ] Documentation updated (README, code comments)
   - [ ] Commit messages follow convention
   - [ ] No merge conflicts with main branch
   - [ ] PR description clearly explains changes

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix (non-breaking change)
- [ ] New feature (non-breaking change)
- [ ] Breaking change (fix or feature that breaks existing functionality)
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Manual testing performed
- [ ] All tests pass

## Related Issues
Fixes #(issue number)

## Screenshots (if applicable)
Add screenshots to help explain your changes

## Checklist
- [ ] My code follows the style guidelines
- [ ] I have performed a self-review
- [ ] I have commented my code where necessary
- [ ] I have updated the documentation
- [ ] My changes generate no new warnings
- [ ] New and existing tests pass
```

## Review Process

### What to Expect

1. **Initial Review**: Within 2-3 business days
2. **Feedback**: Constructive comments and suggestions
3. **Iterations**: You may be asked to make changes
4. **Approval**: At least one maintainer approval required
5. **Merge**: Maintainer will merge once approved

### During Review

- **Be responsive**: Reply to comments and questions promptly
- **Be open**: Consider feedback constructively
- **Ask questions**: If something is unclear, ask!
- **Make changes**: Address all review comments

### After Merge

- Your contribution will be included in the next release
- You'll be added to the contributors list
- Thank you for your contribution! 🎉

## Best Practices

### Do's ✅

- ✅ Write clear, concise commit messages
- ✅ Keep PRs focused on a single change
- ✅ Write tests for your code
- ✅ Update documentation
- ✅ Follow existing code style
- ✅ Be patient and respectful

### Don'ts ❌

- ❌ Submit PRs with failing tests
- ❌ Make unrelated changes in a single PR
- ❌ Ignore review feedback
- ❌ Force push after review has started
- ❌ Include sensitive information (credentials, tokens)
- ❌ Bypass CI/CD checks

## Development Tips

### Debugging

```bash
# Enable verbose logging
./bin/k8s-resource-collector --verbose

# Use Go debugging tools
dlv debug ./cmd/main.go

# Check logs
tail -f /var/log/k8s-resource-collector.log
```

### Performance Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

### Common Issues

**Issue: Tests fail locally but pass in CI**
- Ensure you're using the correct Go version
- Check for environment-specific issues
- Run `make clean` and rebuild

**Issue: Import cycle detected**
- Reorganize code to break circular dependencies
- Consider using interfaces

**Issue: Merge conflicts**
```bash
git fetch upstream
git rebase upstream/main
# Resolve conflicts
git add .
git rebase --continue
```

## Community

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and general discussion
- **Pull Requests**: Code review and collaboration

### Getting Help

- Check existing [documentation](README.md)
- Search [existing issues](https://github.com/midu/k8s-resource-collector/issues)
- Ask in [GitHub Discussions](https://github.com/midu/k8s-resource-collector/discussions)
- Review [examples](examples/) and [tests](tests/)

### Recognition

Contributors will be:
- Added to the contributors list
- Mentioned in release notes (for significant contributions)
- Recognized in the project README

## Additional Resources

- [Go Documentation](https://go.dev/doc/)
- [Kubernetes Client-Go](https://github.com/kubernetes/client-go)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Conventional Commits](https://www.conventionalcommits.org/)

## Questions?

If you have questions about contributing, please:
1. Check this guide thoroughly
2. Search existing issues and discussions
3. Create a new discussion or issue

Thank you for contributing to k8s-resource-collector! Your contributions help make this project better for everyone. 🚀

---

**Last Updated**: November 2025
**Maintainers**: [@midu](https://github.com/midu)

