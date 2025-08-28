# DNS Server Checker

A comprehensive DNS testing tool written in Go that tests multiple DNS servers with various domains and provides detailed categorized reports with real-time progress tracking.

## Features

- **Concurrent DNS Testing**: Tests multiple DNS servers simultaneously for optimal performance
- **Domain Categorization**: Automatically categorizes domains (General, Ad-server, Other, Adult)
- **Real-time Progress**: Interactive progress bar with time estimates and completion tracking
- **Multiple Output Formats**: Support for both JSON and text output
- **Comprehensive Reporting**: Detailed statistics with category-based success rates
- **Configurable Parameters**: Customizable timeout, worker count, and output format
- **Dual Summary Display**: Summary shown both at beginning and end of results

## Configuration

The tool includes predefined configuration constants that can be easily modified:

```go
// Configuration Constants
const (
    DefaultTimeout     = 15          // DNS query timeout in seconds
    DefaultWorkerCount = 50          // Number of concurrent workers
    DefaultFormat     = "text"      // Default output format
    ProgressBarWidth  = 40          // Progress bar width in characters
    ProgressUpdateRate = 100 * time.Millisecond // Progress update frequency
)

// Category Constants
const (
    CategoryGeneral   = "General"
    CategoryAdServer  = "Ad-server"
    CategoryOther     = "Other"
    CategoryAdult     = "Adult"
)
```

## Usage

### Basic Usage

```bash
# Run directly with Go (development)
go run .

# Or use the compiled binary
dns-check-go
```

### CLI Parameters

```bash
# Specify custom DNS servers file
go run . -dns-file custom-servers.txt
dns-check-go -dns-file custom-servers.txt

# Specify custom domains file  
go run . -domains-file custom-domains.txt
dns-check-go -domains-file custom-domains.txt

# Set output format to JSON
go run . -format json
dns-check-go -format json

# Customize timeout and worker count
go run . -timeout 20 -workers 100
dns-check-go -timeout 20 -workers 100

# Save output to file
go run . -output results.txt
dns-check-go -output results.txt

# Show help
go run . -help
dns-check-go -help

# Combine multiple options
go run . -dns-file servers.txt -domains-file domains.txt -format json -timeout 10 -workers 75 -output results.json
dns-check-go -dns-file servers.txt -domains-file domains.txt -format json -timeout 10 -workers 75 -output results.json
```

## Command Line Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `-dns-file` | `dns-servers.txt` | Path to DNS servers list file |
| `-domains-file` | `domains.txt` | Path to domains list file |
| `-format` | `text` | Output format (`text` or `json`) |
| `-timeout` | `15` | DNS query timeout in seconds |
| `-workers` | `50` | Number of concurrent workers |
| `-output` | - | Output file path (optional, prints to stdout if not specified) |

## File Formats

### DNS Servers File (`dns-servers.txt`)

```txt
# Format: IP_ADDRESS DESCRIPTION (optional)
# Lines starting with # are comments
8.8.8.8 Google Public DNS
1.1.1.1 Cloudflare DNS
208.67.222.222 OpenDNS
9.9.9.9 Quad9 DNS
```

### Domains File (`domains.txt`)

```txt
# Format: DOMAIN CATEGORY
# Lines starting with # are comments
google.com General
facebook.com General
youtube.com General
doubleclick.net Ad-server
googlesyndication.com Ad-server
adult-site.xxx Adult
unknown-category.com Other
```

## Domain Categories

- **General**: Common websites and services (google.com, facebook.com, etc.)
- **Ad-server**: Advertisement and tracking domains (doubleclick.net, etc.)
- **Other**: Uncategorized or miscellaneous domains
- **Adult**: Adult content websites

## Progress Tracking

The tool provides real-time progress information:

```bash
Testing DNS servers... ████████████████████████████████████████ 100% (320/320) ETA: 0s
Completed in 12.5s
```

Progress bar features:
- Visual progress indicator with Unicode blocks
- Current/total test count
- Percentage completion
- Estimated time to completion (ETA)
- Total elapsed time

## Installation

### Prerequisites

- Go 1.21 or higher
- Internet connection for DNS queries

### Method 1: Clone and Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd dns-check-go

# Install dependencies
go mod tidy

# Run directly with Go (development)
go run .
```

### Method 2: Build Binary

```bash
# Clone repository
git clone <repository-url>
cd dns-check-go

# Initialize Go module (if needed)
go mod init dns-check-go

# Install dependencies
go get github.com/miekg/dns

# Build the executable
go build -o dns-check-go main.go

# Run the binary
./dns-check-go
```

### Dependencies

```bash
go get github.com/miekg/dns
```

## Building

```bash
# Build for current platform
go build -o dns-check-go main.go

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o dns-check-go.exe main.go

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o dns-check-go-linux main.go

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -o dns-check-go-mac main.go
```

## Performance Notes

- **Concurrent Processing**: Uses worker pool pattern with configurable concurrent workers
- **Progress Updates**: Progress bar updates every 100ms to minimize overhead
- **Memory Efficiency**: Optimized memory usage with streaming results
- **Scalable**: Default configuration supports up to 50 concurrent workers
- **Fast Execution**: Typical test of 4 DNS servers × 20 domains completes in 5-15 seconds

## License

This project is open source. Feel free to use, modify, and distribute according to your needs.

## Contributing

We welcome contributions to improve the DNS Server Checker! Here's how you can contribute:

### Getting Started

1. **Fork the Repository**: Create your own fork of the project
2. **Clone Your Fork**: `git clone https://github.com/your-username/dns-check-go.git`
3. **Create a Branch**: `git checkout -b feature/your-feature-name`

### Development Setup

```bash
# Clone your fork
git clone https://github.com/your-username/dns-check-go.git
cd dns-check-go

# Install dependencies
go mod tidy

# Run tests (when available)
go test ./...

# Run the tool for testing
go run . -dns-file dns-servers.txt -domains-file domains.txt
```

### Types of Contributions

- **🐛 Bug Reports**: Found a bug? Please create an issue with detailed information
- **💡 Feature Requests**: Have ideas for new features? We'd love to hear them!
- **📖 Documentation**: Improve README, comments, or add examples
- **🔧 Code Improvements**: Performance optimizations, code refactoring
- **🌍 Translations**: Add support for more languages
- **🧪 Testing**: Add unit tests, integration tests, or test with different configurations

### Code Guidelines

- Follow Go best practices and conventions
- Keep functions focused and well-documented
- Use meaningful variable and function names
- Add comments for complex logic
- Ensure backward compatibility when possible

### Submitting Changes

1. **Test Your Changes**: Ensure your code works with various DNS servers and domain lists
2. **Update Documentation**: Update README if you've added new features
3. **Commit Your Changes**: Use descriptive commit messages
   ```bash
   git add .
   git commit -m "Add feature: DNS over HTTPS support"
   ```
4. **Push to Your Fork**: `git push origin feature/your-feature-name`
5. **Create Pull Request**: Submit a PR with detailed description of changes

### Pull Request Guidelines

- **Clear Title**: Use descriptive titles (e.g., "Fix timeout handling for large domain lists")
- **Detailed Description**: Explain what changes you made and why
- **Test Results**: Include test results or screenshots if applicable
- **Link Issues**: Reference any related issues using `#issue-number`

## Support

If you encounter any issues or have questions:

### Reporting Issues

1. **Check Existing Issues**: Search [existing issues](https://github.com/username/dns-check-go/issues) first
2. **Create Detailed Reports**: When creating new issues, please include:
   - **System Information**: OS, Go version, system architecture
   - **Command Used**: The exact command that caused the issue
   - **Expected Behavior**: What you expected to happen
   - **Actual Behavior**: What actually happened
   - **Error Messages**: Any error messages or logs
   - **Sample Files**: If relevant, include your DNS servers or domains files

### Issue Templates

**Bug Report Example:**
```
**System Information:**
- OS: Windows 11 / Ubuntu 22.04 / macOS 13
- Go Version: 1.21.5
- Architecture: amd64

**Command Used:**
`dns-check-go -dns-file custom-servers.txt -timeout 5 -workers 100`

**Expected Behavior:**
Tool should complete testing all domains within specified timeout

**Actual Behavior:**
Tool hangs at 50% progress and doesn't complete

**Error Messages:**
[Include any error output here]

**Additional Context:**
- Custom DNS file contains 10 servers
- Testing with 50 domains
- Issue occurs consistently
```

### Getting Help

- **📚 Documentation**: Check this README for comprehensive usage information
- **🐛 Bug Reports**: Create an issue for bugs or unexpected behavior
- **💬 Feature Requests**: Suggest new features via GitHub issues
- **❓ Questions**: For general questions, create a discussion or issue

### Community Guidelines

- Be respectful and constructive in all interactions
- Provide detailed information when reporting issues
- Help others by sharing your knowledge and experience
- Test thoroughly before submitting contributions
