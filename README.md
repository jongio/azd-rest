# azd-rest

Azure Developer CLI extension for sending REST API calls with automatic Azure authentication and scope detection. This repository follows the azd-exec patterns and starts with project scaffolding for future implementation.

## Status

- Scaffolding in place; CLI implementation will be added in follow-on tasks.

## Prerequisites

- Go 1.25.5+
- Node.js 20+ and pnpm
- Azure Developer CLI with extensions enabled (`azd config set alpha.extension.enabled on`)

## Getting Started (local development)

1. Clone this repository and switch to the `azd-rest` directory.
2. Install Go dependencies: `cd cli && go mod download`.
3. Run the placeholder test script: `pnpm test` (will exercise CLI tests as they are added).

## Local installation (once builds are produced)

- Build and pack will mirror the azd-exec flow (`azd x build` / `azd x pack`).
- After packing, install locally:
  - `azd extension install jongio.azd.rest --source local --path <path-to-packed-artifacts>`

## License

Licensed under the MIT License. See LICENSE for details.
