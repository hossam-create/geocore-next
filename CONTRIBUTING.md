# Contributing to GeoCore Next

Thank you for your interest in contributing!

## Development Setup
1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/geocore-next`
3. Follow the Quick Start in README.md

## Branch Naming
- Feature: `feat/description` (e.g. `feat/dutch-auction`)
- Fix: `fix/description` (e.g. `fix/websocket-reconnect`)
- Docs: `docs/description`

## Commit Convention (Conventional Commits)
- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation
- `refactor:` code refactor
- `test:` adding tests
- `chore:` tooling/config

Example: `feat: add Dutch auction type with auto price decrease`

## Pull Request Process
1. Create a branch from `main`
2. Make your changes with tests
3. Run `go test ./...` and `npm run test`
4. Submit PR with a clear description
5. Link any related issues

## Code Style
- Go: follow `gofmt` and `go vet`
- TypeScript: ESLint + Prettier (configs included)
