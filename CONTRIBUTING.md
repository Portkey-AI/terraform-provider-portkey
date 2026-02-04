# Contributing to Portkey Terraform Provider

Thank you for your interest in contributing to the Portkey Terraform Provider! We welcome contributions from the community.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and collaborative environment.

## How to Contribute

### Reporting Issues

- **Search First**: Check if the issue already exists in the [issue tracker](https://github.com/Portkey-AI/terraform-provider-portkey/issues)
- **Provide Details**: Include as much information as possible:
  - Provider version
  - Terraform version
  - Go version (if building from source)
  - Steps to reproduce
  - Expected vs actual behavior
  - Relevant configuration files (sanitized)
  - Error messages and logs

### Suggesting Features

- Open an issue with the `enhancement` label
- Describe the use case and benefits
- Provide examples of how it would work

### Submitting Pull Requests

1. **Fork and Clone**
```bash
git clone https://github.com/YOUR-USERNAME/terraform
cd terraform
```

2. **Create a Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make Changes**
   - Write clear, maintainable code
   - Follow Go best practices
   - Add tests for new functionality
   - Update documentation

4. **Test Your Changes**
   ```bash
   # Run unit tests
   go test ./...
   
   # Build the provider
   make build
   
   # Install locally and test
   make install
   ```

5. **Commit and Push**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   git push origin feature/your-feature-name
   ```

6. **Open Pull Request**
   - Provide a clear description of changes
   - Reference any related issues
   - Ensure CI checks pass

## Development Setup

### Prerequisites

- Go 1.21 or later
- Terraform 1.0 or later
- Portkey Admin API key for testing

### Building from Source

```bash
# Clone the repository
git clone https://github.com/Portkey-AI/terraform-provider-portkey
cd terraform

# Install dependencies
go mod download

# Build the provider
make build

# Install locally
make install
```

### Running Tests

```bash
# Unit tests
go test ./...

# Acceptance tests (requires valid API key)
export PORTKEY_API_KEY="your-admin-api-key"
make testacc
```

## Code Style

### Go Code

- Follow standard Go formatting: `gofmt` and `go vet`
- Use meaningful variable and function names
- Add comments for complex logic
- Keep functions focused and concise

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Test additions or changes
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

Examples:
```
feat: add virtual_key resource
fix: handle nil pointer in workspace update
docs: update installation instructions
```

## Project Structure

```
terraform/
├── internal/
│   ├── client/          # API client implementation
│   └── provider/        # Terraform resources and data sources
├── examples/            # Usage examples
├── docs/               # Documentation
└── main.go             # Provider entry point
```

### Adding a New Resource

1. Create `{resource_name}_resource.go` in `internal/provider/`
2. Implement the resource interface:
   - Schema definition
   - Create, Read, Update, Delete operations
   - Import functionality
3. Register the resource in `provider.go`
4. Add tests
5. Update documentation

### Adding a New Data Source

1. Create `{resource_name}_data_source.go` in `internal/provider/`
2. Implement the data source interface:
   - Schema definition
   - Read operation
3. Register the data source in `provider.go`
4. Add tests
5. Update documentation

## Testing Guidelines

### Unit Tests

- Test individual functions and methods
- Mock external dependencies
- Cover edge cases and error conditions

### Acceptance Tests

- Test full resource lifecycle (Create, Read, Update, Delete)
- Test import functionality
- Use unique resource names to avoid conflicts
- Clean up resources after tests

Example:
```go
func TestAccWorkspaceResource(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Create and Read testing
            {
                Config: testAccWorkspaceConfig("test-name"),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("portkey_workspace.test", "name", "test-name"),
                ),
            },
            // Import testing
            {
                ResourceName:      "portkey_workspace.test",
                ImportState:       true,
                ImportStateVerify: true,
            },
        },
    })
}
```

## Documentation

- Update README.md for significant changes
- Add examples for new resources
- Document breaking changes
- Keep CHANGELOG.md updated

## PR Review Guidelines (Maintainers)

This section outlines the standard process for reviewing, merging, and releasing PRs in this repository.

### Phase 1: Initial Assessment

```bash
# View open PRs
gh pr list

# Checkout the PR locally
gh pr checkout <PR-number>
```

**CHECKPOINT 1 - Understand the PR:**
- Read the PR description and linked issues
- Ask yourself: *What problem does this solve? Is this functionality we need?*
- For new features: verify we don't already cover this functionality elsewhere

### Phase 2: Code Review

Run these checks locally:

```bash
# Build the provider
go build ./...

# Run static analysis
go vet ./...
gofmt -d .

# Run linter (note: may have known issues with golangci-lint v2)
golangci-lint run

# Run the full test suite
make testacc
```

**CHECKPOINT 2 - Evaluate code quality against these criteria:**

#### 2.1 Test Quality (No Malicious Compliance)
- [ ] Tests cover the full CRUD lifecycle (Create, Read, Update, Delete)
- [ ] Tests verify import functionality works
- [ ] Tests check actual attribute values, not just existence
- [ ] Tests cover edge cases (updates, deletions, error conditions)
- [ ] Tests are meaningful, not just "passes but tests nothing"

Example of a **good** test check:
```go
resource.TestCheckResourceAttr("portkey_resource.test", "name", "expected-value"),
```

Example of **malicious compliance** (avoid):
```go
// Only checks attribute exists, not its value
resource.TestCheckResourceAttrSet("portkey_resource.test", "id"),
```

#### 2.2 Impact on Existing Code
- [ ] Changes are additive (new files) vs. modifying existing logic
- [ ] If modifying existing code: understand why and verify no regressions
- [ ] Check `provider.go` changes - should only add new registrations
- [ ] Verify no unrelated changes snuck in

#### 2.3 Code Style Consistency
- [ ] File naming follows pattern: `{resource}_resource.go`, `{resource}_data_source.go`
- [ ] Schema definitions match existing resources (descriptions, validators, plan modifiers)
- [ ] Error handling is consistent with other resources
- [ ] API client methods follow existing patterns in `client/client.go`

#### 2.4 Documentation
- [ ] Resource docs exist in `docs/resources/` or `docs/data-sources/`
- [ ] Examples are provided
- [ ] RESOURCE_MATRIX.md is updated (if applicable)

### Phase 3: Provide Feedback

**If changes are needed:**

```bash
gh pr review <PR-number> --request-changes --body "Your detailed feedback"
```

Be specific about:
- What needs to change and why
- Provide code examples when helpful
- Reference existing code patterns to follow

**If PR is good:**

```bash
gh pr review <PR-number> --approve --body "Your approval message"
```

**CHECKPOINT 3 - Before approving, verify:**
- [ ] CI checks are passing (or failures are pre-existing/unrelated)
- [ ] You've run tests locally
- [ ] Code compiles without errors
- [ ] You understand what the code does

### Phase 4: Merge

```bash
# Merge the PR
gh pr merge <PR-number> --merge

# Pull changes to local main
git checkout main && git pull
```

### Phase 5: Update CHANGELOG

Edit `CHANGELOG.md`:
1. Add entry under `[Unreleased]` describing what was added/changed/fixed
2. Follow the existing format and categorization (Added, Changed, Fixed, etc.)

### Phase 6: Release

When ready to release (can batch multiple PRs):

```bash
# 1. Update CHANGELOG.md
#    - Change [Unreleased] to [X.Y.Z] - YYYY-MM-DD
#    - Add new empty [Unreleased] section
#    - Update comparison links at bottom

# 2. Commit and push
git add CHANGELOG.md
git commit -m "chore: update CHANGELOG for vX.Y.Z"
git push origin main

# 3. Create and push tag
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z

# 4. Monitor release
gh run watch $(gh run list --workflow=release.yml --limit 1 --json databaseId -q '.[0].databaseId') --exit-status

# 5. Verify
gh release view vX.Y.Z
```

### Troubleshooting

#### CI Lint Failures
The `golangci-lint` configuration may have compatibility issues with certain Go versions. If lint fails in CI but the PR code is correct:
- Verify the lint errors are pre-existing (not introduced by the PR)
- Check if errors are in files not touched by the PR
- The lint config (`.golangci.yml`) may need updates when Go version changes

#### Force Push After Rebase
If rebasing a PR branch and `git push --force-with-lease` fails with "stale info":
```bash
git push --force  # Use with caution, only on PR branches
```

#### CI Flakiness
If CI fails intermittently:
- Re-run the workflow: `gh run rerun <run-id>`
- Check if it's a timing/race condition in tests
- Check if it's an API rate limit issue

#### Contributor Can't Push Fixes
If a contributor needs help making changes and "Allow edits from maintainers" is enabled:
```bash
# Add their fork as remote
git remote add contributor-fork https://github.com/CONTRIBUTOR/terraform-provider-portkey.git

# Push fixes to their branch
git push contributor-fork HEAD:their-branch-name
```

### Review Mindset

Every PR review is a learning opportunity. After each review session, take a moment to reflect:

- Did you learn a new pattern or technique from the contributor's code?
- Did you discover an edge case you hadn't considered before?
- Did the discussion reveal gaps in documentation or testing patterns?
- Could this PR's approach improve other parts of the codebase?

Good feedback flows both ways. Stay curious, ask questions when something is unclear, and remember that even experienced contributors can learn something new from every code review.

## Releasing

### Pre-release Checklist

1. **Merge all PRs** for the release
2. **Verify CI passes** on main branch

### Release Steps

```bash
# 1. Checkout and pull latest main
git checkout main && git pull

# 2. Update CHANGELOG.md
#    - Change [Unreleased] to [X.Y.Z] with today's date
#    - Add new empty [Unreleased] section
#    - Update comparison links at bottom of file

# 3. Commit changelog
git add CHANGELOG.md
git commit -m "chore: update CHANGELOG for vX.Y.Z"
git push origin main

# 4. Create and push tag (triggers release workflow)
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin vX.Y.Z

# 5. Monitor release workflow
gh run list --workflow=release.yml --limit 1
gh run watch <run-id> --exit-status

# 6. Verify release
gh release view vX.Y.Z
```

### Version Numbering

Follow [Semantic Versioning](https://semver.org/):
- **MAJOR** (1.0.0): Breaking changes
- **MINOR** (0.X.0): New features, backwards compatible
- **PATCH** (0.0.X): Bug fixes, backwards compatible

### What the Release Workflow Does

The GitHub Action (`.github/workflows/release.yml`) automatically:
1. Builds binaries for all platforms (darwin, linux, windows)
2. Signs artifacts with GPG
3. Creates GitHub release with changelog
4. Publishes to Terraform Registry

## Getting Help

- **Questions**: Open a [GitHub Discussion](https://github.com/Portkey-AI/terraform-provider-portkey/discussions)
- **Bugs**: File an [issue](https://github.com/Portkey-AI/terraform-provider-portkey/issues)
- **Chat**: Join the [Portkey Discord](https://portkey.sh/discord-1)

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (Mozilla Public License 2.0).

## Thank You!

Your contributions help make this project better for everyone! 🙏

