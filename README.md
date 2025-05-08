# parta

A Go module project

## Project Structure

```
.
├── cmd/          # Main applications
│   └── multiProcess_v3.go
├── pkg/          # Library code
├── go.mod        # Go module definition
├── go.sum        # Dependency checksums
├── LICENSE       # License file
└── README.md     # Project documentation
```

## Getting Started

1. Clone the repository
2. Set GOPROXY for China:
   ```bash
   go env -w GOPROXY=https://goproxy.cn,direct
   ```
3. Install dependencies:
   ```bash
   go mod tidy
   ```
4. Build and run:
   ```bash
   go build ./cmd/...
   ```
