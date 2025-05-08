# parta

parallel task

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

## update

`git add -A  && git commit -m "v1.0.15" && git push && git tag v1.0.15 && git push origin v1.0.15`

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
