# Automated Release Process

This document describes the automated release process for the DigitalOcean Firewall Allowlister project, which uses conventional commits and semantic versioning to automatically manage releases.

## Overview

The project uses a dual-workflow system:

1. **Automated Pre-Releases**: Triggered automatically on pushes to the `main` branch
2. **Manual Full Releases**: Triggered manually by maintainers through GitHub Actions

## Conventional Commits

All commits must follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Supported Types

- `feat`: A new feature (triggers minor version bump)
- `fix`: A bug fix (triggers patch version bump)
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance (triggers patch version bump)
- `test`: Adding missing tests or correcting existing tests
- `build`: Changes that affect the build system or external dependencies
- `ci`: Changes to CI configuration files and scripts
- `chore`: Other changes that don't modify src or test files
- `revert`: Reverts a previous commit

### Breaking Changes

To trigger a major version bump, use one of these patterns:

1. Add `!` after the type: `feat!: remove deprecated API`
2. Add `BREAKING CHANGE:` in the footer:

   ```
   feat: add new authentication method

   BREAKING CHANGE: The old authentication method is no longer supported
   ```

### Examples

```bash
# Patch version bump (1.0.0 -> 1.0.1)
git commit -m "fix: resolve memory leak in connection pool"

# Minor version bump (1.0.0 -> 1.1.0)
git commit -m "feat: add support for IPv6 addresses"

# Major version bump (1.0.0 -> 2.0.0)
git commit -m "feat!: redesign configuration format"

# No version bump (documentation, tests, etc.)
git commit -m "docs: update installation instructions"
```

## Release Types

### Pre-Releases (Automated)

- **Trigger**: Push to `main` branch
- **Version Format**: `v1.2.3-pre.TIMESTAMP`
- **Behavior**:
  - Automatically calculates semantic version based on conventional commits
  - Creates a pre-release tag
  - Builds and publishes Docker images
  - Creates GitHub release marked as pre-release
  - Updates CHANGELOG.md

### Full Releases (Manual)

- **Trigger**: Manual workflow dispatch or pushing a version tag
- **Version Format**: `v1.2.3`
- **Behavior**:
  - Can be triggered from GitHub Actions UI
  - Creates a stable release tag
  - Builds and publishes Docker images with `latest` tag
  - Creates GitHub release marked as stable
  - Updates CHANGELOG.md

## How to Create Releases

### Automatic Pre-Release

1. Make changes following conventional commit format
2. Push to `main` branch
3. GitHub Actions automatically:
   - Calculates next version
   - Creates pre-release
   - Publishes artifacts

### Manual Full Release

1. Go to GitHub Actions tab
2. Select "Release" workflow
3. Click "Run workflow"
4. Choose options:
   - **Release Type**: `release` for stable, `prerelease` for pre-release
   - **Version Override**: Optional version override (e.g., `v1.2.3`)
5. Click "Run workflow"

### Tag-Based Release

Push a semantic version tag:

```bash
git tag v1.2.3
git push origin v1.2.3
```

## Validation and Quality Checks

The release process includes several validation steps:

1. **PR Validation**:

   - PR titles must follow conventional commit format
   - Commit messages are validated using commitlint

2. **Release Validation**:

   - Semantic version format validation
   - Additional checks for manual releases

3. **Testing**:
   - All tests must pass before release
   - Code coverage is reported

## Docker Images

Multi-architecture Docker images are automatically built and published to:

- **GitHub Container Registry**: `ghcr.io/kholisrag/do-firewall-allowlister`
- **Docker Hub**: `kholisrag/do-firewall-allowlister`
- **Quay.io**: `quay.io/kholisrag/do-firewall-allowlister`

### Image Tags

- `latest`: Latest stable release
- `v1.2.3`: Specific version
- `v1.2.3-pre.TIMESTAMP`: Pre-release versions

## Changelog

The `CHANGELOG.md` file is automatically generated and categorized:

- 🚀 **Features**: New functionality
- 🐛 **Bug Fixes**: Bug fixes
- ⚡ **Performance Improvements**: Performance enhancements
- 💥 **Breaking Changes**: Breaking changes
- 🔧 **Build & CI**: Build and CI changes
- 📚 **Documentation**: Documentation updates
- 🧪 **Tests**: Test additions/changes
- 🔄 **Reverts**: Reverted changes
- 📦 **Dependencies**: Dependency updates
- 🏗️ **Others**: Other changes

## Troubleshooting

### Common Issues

1. **Invalid commit message format**

   - Error: PR validation fails
   - Solution: Follow conventional commit format

2. **Version calculation fails**

   - Error: Semantic version action fails
   - Solution: Ensure there are conventional commits since last tag

3. **Docker build fails**
   - Error: GoReleaser fails during Docker build
   - Solution: Check Dockerfile and build context

### Getting Help

- Check the [GitHub Actions logs](https://github.com/kholisrag/do-firewall-allowlister/actions) for detailed error messages
- Review the [Conventional Commits specification](https://www.conventionalcommits.org/)
- Open an issue if you encounter problems with the release process

## Configuration Files

- `.github/workflows/release.yaml`: Main release workflow
- `.github/workflows/pr-validation.yaml`: PR validation workflow
- `.goreleaser.yaml`: GoReleaser configuration
- `.commitlintrc.json`: Commit message validation rules
