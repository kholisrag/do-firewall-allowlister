# Documentation

This directory contains documentation for the DigitalOcean Firewall Allowlister project.

## Contents

- [**RELEASE_PROCESS.md**](RELEASE_PROCESS.md) - Comprehensive guide to the automated release process, conventional commits, and semantic versioning

## Quick Links

- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GoReleaser Documentation](https://goreleaser.com/)

## Contributing

When contributing to this project, please:

1. Follow the [conventional commit format](RELEASE_PROCESS.md#conventional-commits)
2. Ensure your PR title follows the same format
3. Update documentation as needed
4. Add tests for new features

## Release Process Summary

1. **Development**: Make changes using conventional commits
2. **Pre-Release**: Push to `main` → automatic pre-release
3. **Full Release**: Manual trigger through GitHub Actions
4. **Distribution**: Multi-platform Docker images and binaries

For detailed information, see [RELEASE_PROCESS.md](RELEASE_PROCESS.md).
