# Building PanickedBot

This guide provides detailed instructions for building PanickedBot from source.

## Quick Start

```bash
# One-command build
make

# Or step by step
make generate  # Generate sqlc code
make build     # Build binary
```

## Requirements

- Go 1.25 or later
- sqlc (installed via `make install-tools`)

## Build Process

PanickedBot uses [sqlc](https://sqlc.dev/) to generate type-safe Go code from SQL queries. The generated code is **NOT** committed to version control and must be generated as part of the build process.

### Step 1: Install Tools

```bash
make install-tools
```

This installs sqlc to `$(go env GOPATH)/bin`. Make sure this directory is in your `PATH`.

### Step 2: Generate Database Code

```bash
make generate
```

This runs `sqlc generate` which:
- Reads SQL queries from `internal/db/queries/*.sql`
- Reads the database schema from `schema.sql`
- Generates type-safe Go code in `internal/db/sqlc/`

### Step 3: Build

```bash
make build
```

This compiles the Go code and produces the `PanickedBot` binary.

## Makefile Targets

Run `make help` to see all available targets:

```
make all           # Generate sqlc code and build (default)
make generate      # Generate sqlc code from SQL queries
make build         # Build the binary
make clean         # Remove generated files and binaries
make test          # Run tests
make vet           # Run go vet
make install-tools # Install required tools (sqlc)
make help          # Display help message
```

## Manual Build

If you prefer not to use Make:

```bash
# Install sqlc
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Generate code
sqlc generate

# Build
go build -o PanickedBot .
```

## Continuous Integration

The GitHub Actions workflow automatically:
1. Installs sqlc
2. Generates database code
3. Builds the project
4. Runs tests and static analysis

**Important:** For pull requests, the CI tests the merge commit (the result of merging the PR branch into the target branch). This ensures that the code will build successfully after merging and catches integration issues early.

See `.github/workflows/build.yml` for details.

## Troubleshooting

### "sqlc: command not found"

Make sure `$(go env GOPATH)/bin` is in your PATH:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

Or add to your shell profile (~/.bashrc, ~/.zshrc, etc.):

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Generated files are missing

Run `make generate` to create the generated files in `internal/db/sqlc/`.

### Build fails with import errors

Make sure you've run `make generate` first. The generated code must exist before building.

## Development Workflow

1. Make changes to SQL queries in `internal/db/queries/`
2. Run `make generate` to regenerate Go code
3. Make changes to Go code as needed
4. Run `make build` to compile
5. Test your changes
6. **Do NOT commit** files in `internal/db/sqlc/` - they are generated during build

## Why Generated Code Is Not Committed

Generated code is excluded from version control because:

1. **Source of Truth**: SQL queries in `internal/db/queries/` are the source of truth
2. **Reduces Merge Conflicts**: Generated code can conflict during merges
3. **Smaller Repository**: Reduces repository size
4. **Always Fresh**: CI always builds with latest generated code
5. **Best Practice**: Standard practice for code generation tools

The CI workflow ensures that the code can always be generated and built successfully.
