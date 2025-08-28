package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

// Configuration constants
const (
	DefaultTimeout     = 15                     // Default DNS query timeout in seconds
	DefaultWorkerCount = 50                     // Default number of concurrent workers
	DefaultFormat      = "text"                 // Default output format
	ProgressBarWidth   = 40                     // Progress bar width in characters
	ProgressUpdateRate = 100 * time.Millisecond // Progress bar update frequency
)

// Categories for domain classification
var (
	CategoryGeneral  = "General"
	CategoryAdServer = "Ad-server"
	CategoryOther    = "Other"
	CategoryAdult    = "Adult"
	CategoryOrder    = []string{CategoryGeneral, CategoryAdServer, CategoryOther, CategoryAdult}
)

// DNSServer represents a DNS server
type DNSServer struct {
	IP          string `json:"ip"`
	Description string `json:"description,omitempty"`
}

// TestResult represents the result of a DNS test
type TestResult struct {
	Server       DNSServer     `json:"server"`
	Domain       string        `json:"domain"`
	Category     string        `json:"category"`
	Success      bool          `json:"success"`
	ResponseTime time.Duration `json:"response_time_ms"`
	IP           string        `json:"resolved_ip,omitempty"`
	Error        string        `json:"error,omitempty"`
}

// TestResults represents all test results
type TestResults struct {
	Timestamp time.Time    `json:"timestamp"`
	Results   []TestResult `json:"results"`
	Summary   Summary      `json:"summary"`
}

// DomainCategory represents a domain with its category
type DomainCategory struct {
	Domain   string
	Category string
}

// CategoryStats represents statistics for a category
type CategoryStats struct {
	TotalTests      int     `json:"total_tests"`
	SuccessfulTests int     `json:"successful_tests"`
	FailedTests     int     `json:"failed_tests"`
	SuccessRate     float64 `json:"success_rate"`
}

// Summary represents test summary
type Summary struct {
	TotalTests          int                      `json:"total_tests"`
	SuccessfulTests     int                      `json:"successful_tests"`
	FailedTests         int                      `json:"failed_tests"`
	SuccessRate         float64                  `json:"success_rate"`
	AverageResponseTime time.Duration            `json:"average_response_time_ms"`
	CategoryStats       map[string]CategoryStats `json:"category_stats"`
}

// Default test domains with categories
var defaultDomains = []DomainCategory{
	// General websites
	{"google.com", CategoryGeneral},
	{"youtube.com", CategoryGeneral},
	{"facebook.com", CategoryGeneral},
	{"instagram.com", CategoryGeneral},
	{"twitter.com", CategoryGeneral},
	{"x.com", CategoryGeneral},
	{"github.com", CategoryGeneral},
	{"stackoverflow.com", CategoryGeneral},
	{"reddit.com", CategoryGeneral},
	{"netflix.com", CategoryGeneral},
	{"amazon.com", CategoryGeneral},
	{"microsoft.com", CategoryGeneral},
	{"apple.com", CategoryGeneral},
	{"cloudflare.com", CategoryGeneral},
	{"wikipedia.org", CategoryGeneral},
	{"yandex.com", CategoryGeneral},
	{"baidu.com", CategoryGeneral},

	// Other services
	{"pastebin.com", CategoryOther},

	// Adult content
	{"pornhub.com", CategoryAdult},
	{"xvideos.com", CategoryAdult},

	// Advertisement and tracking servers
	{"googleadservices.com", CategoryAdServer},
	{"googlesyndication.com", CategoryAdServer},
	{"googletagmanager.com", CategoryAdServer},
	{"doubleclick.net", CategoryAdServer},
	{"google-analytics.com", CategoryAdServer},
	{"adsystem.amazon.com", CategoryAdServer},
	{"amazon-adsystem.com", CategoryAdServer},
	{"connect.facebook.net", CategoryAdServer},
	{"ads.linkedin.com", CategoryAdServer},
	{"analytics.twitter.com", CategoryAdServer},
	{"ads.twitter.com", CategoryAdServer},
	{"ads.yahoo.com", CategoryAdServer},
	{"advertising.com", CategoryAdServer},
	{"adsystem.microsoft.com", CategoryAdServer},
	{"bat.bing.com", CategoryAdServer},
}

// Default DNS servers
/*var defaultDNSServers = []DNSServer{
	{"8.8.8.8", "Google DNS"},
	{"8.8.4.4", "Google DNS Secondary"},
	{"1.1.1.1", "Cloudflare DNS"},
	{"1.0.0.1", "Cloudflare DNS Secondary"},
	{"208.67.222.222", "OpenDNS"},
	{"208.67.220.220", "OpenDNS Secondary"},
	{"9.9.9.9", "Quad9 DNS"},
	{"149.112.112.112", "Quad9 DNS Secondary"},
}*/

var defaultDNSServers = []DNSServer{
	{"212.154.100.18", "TR - Türknet"},
	{"193.192.98.8", "TR - Türknet Secondary"},
	{"1.1.1.1", "AU - Cloudflare"},
	{"1.0.0.1", "AU - Cloudflare Secondary"},
	{"45.90.28.230", "US - NextDNS"},
	{"45.90.30.230", "US - NextDNS Secondary"},
	{"8.8.4.4", "US - Google Public DNS"},
	{"8.8.8.8", "US - Google Public DNS Secondary"},
	{"92.45.23.168", "TR - deik.org.tr"},
	{"195.244.44.45", "TR - CubeDNS - Netinternet"},
	{"195.244.44.44", "TR - CubeDNS - Netinternet Secondary"},
	{"9.9.9.9", "US - Quad9 Security"},
	{"149.112.112.112", "US - Quad9 Security Secondary"},
	{"149.112.112.10", "US - Quad9 No Security"},
	{"9.9.9.10", "US - Quad9 No Security Secondary"},
	{"156.154.71.1", "US - Neustar 1"},
	{"156.154.70.1", "US - Neustar 1 Secondary"},
	{"209.244.0.3", "US - Level 3 - A"},
	{"209.244.0.4", "US - Level 3 - A Secondary"},
	{"4.2.2.1", "US - Level 3 - B"},
	{"4.2.2.2", "US - Level 3 - B Secondary"},
	{"4.2.2.3", "US - Level 3 - C"},
	{"4.2.2.4", "US - Level 3 - C Secondary"},
	{"4.2.2.5", "US - Level 3 - D"},
	{"4.2.2.6", "US - Level 3 - D Secondary"},
	{"204.69.234.1", "US - UltraDNS"},
	{"204.74.101.1", "US - UltraDNS Secondary"},
	{"156.154.70.5", "US - Neustar 2"},
	{"156.154.71.5", "US - Neustar 2 Secondary"},
	{"199.85.126.10", "US - Norton ConnectSafe"},
	{"199.85.127.10", "US - Norton ConnectSafe Secondary"},
	{"198.153.192.1", "US - Norton DNS"},
	{"198.153.194.1", "US - Norton DNS Secondary"},
	{"64.6.65.6", "US - VeriSign Public DNS"},
	{"64.6.64.6", "US - VeriSign Public DNS Secondary"},
	{"156.154.71.22", "US - Comodo"},
	{"156.154.70.22", "US - Comodo Secondary"},
	{"208.67.220.220", "US - OpenDNS"},
	{"208.67.222.222", "US - OpenDNS Secondary"},
	{"208.67.222.220", "US - OpenDNS - 2"},
	{"195.46.39.39", "RU - Safe DNS"},
	{"195.46.39.40", "RU - Safe DNS Secondary"},
	{"176.9.1.117", "DE - DNSForge - Normal"},
	{"176.9.93.198", "DE - DNSForge - Normal Secondary"},
	{"49.12.223.2", "DE - DNSForge - Clean"},
	{"49.12.43.208", "DE - DNSForge - Clean Secondary"},
	{"195.92.195.94", "GB - Orange DNS"},
	{"195.92.195.95", "GB - Orange DNS Secondary"},
	{"49.12.222.213", "DE - DNSForge - Hard"},
	{"88.198.122.154", "DE - DNSForge - Hard Secondary"},
	{"138.199.149.249", "DE - DNSForge - Blank"},
	{"78.47.71.194", "DE - DNSForge - Blank Secondary"},
	{"163.172.141.219", "90dns - FR - US"},
	{"207.246.121.77", "90dns - FR - US Secondary"},
	{"185.228.169.9", "CleanBrowsing"},
	{"185.228.168.9", "CleanBrowsing Secondary"},
	{"8.26.56.26", "US - Comodo Secure"},
	{"8.20.247.20", "US - Comodo Secure Secondary"},
	{"8.20.247.10", "US - Comodo Secure Filtering"},
	{"8.26.56.10", "US - Comodo Secure Filtering Secondary"},
	{"212.23.8.1", "GB - Zen Internet"},
	{"212.23.3.1", "GB - Zen Internet Secondary"},
	{"94.140.15.15", "RU - AdGuard DNS"},
	{"94.140.14.14", "RU - AdGuard DNS Secondary"},
	{"74.82.42.42", "US - Hurricane Electric"},
	{"77.88.8.1", "RU - Yandex"},
	{"77.88.8.8", "RU - Yandex Secondary"},
	{"205.171.2.65", "US - Qwest"},
	{"205.171.3.65", "US - Qwest Secondary"},
	{"80.80.80.80", "NL - Freenom World"},
	{"80.80.81.81", "NL - Freenom World Secondary"},
	{"216.146.36.36", "US - Dyn"},
	{"216.146.35.35", "US - Dyn Secondary"},
	{"95.216.149.205", "LavaDNS - dns.lavate.ch"},
	{"46.20.159.27", "TR - Dora Telekom"},
	{"46.20.159.27", "TR - Dora Telekom Secondary"},
	{"76.76.19.19", "Alternate DNS"},
	{"76.223.122.150", "Alternate DNS Secondary"},
	{"89.233.43.71", "DK - Censurfridns"},
	{"91.239.100.100", "DK - Censurfridns Secondary"},
	{"80.67.169.12", "FR - FDN"},
	{"80.67.169.40", "FR - FDN Secondary"},
	{"199.2.252.10", "US - Sprintlink"},
	{"204.97.212.10", "US - Sprintlink Secondary"},
	{"84.200.69.80", "DE - DNS WATCH"},
	{"84.200.70.40", "DE - DNS WATCH Secondary"},
	{"204.97.212.10", "US - Sprint"},
	{"204.117.214.10", "US - Sprint Secondary"},
}

func main() {
	var (
		listFile    = flag.String("list", "", "DNS server list file (optional)")
		outputFile  = flag.String("output", "", "Output file for results (optional, defaults to stdout)")
		helpFlag    = flag.Bool("help", false, "Show help")
		formatFlag  = flag.String("format", DefaultFormat, "Output format: json, text")
		timeoutFlag = flag.Int("timeout", DefaultTimeout, "Timeout in seconds for DNS queries")
		workersFlag = flag.Int("workers", DefaultWorkerCount, "Number of concurrent workers")
	)

	flag.Parse()

	if *helpFlag {
		printHelp()
		return
	}

	// Load DNS servers
	var dnsServers []DNSServer
	if *listFile != "" {
		servers, err := loadDNSServersFromFile(*listFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading DNS servers from file: %v\n", err)
			os.Exit(1)
		}
		dnsServers = servers
	} else {
		dnsServers = defaultDNSServers
		fmt.Fprintf(os.Stderr, "Using default DNS servers list\n")
	}

	fmt.Fprintf(os.Stderr, "Testing %d DNS servers against %d domains...\n", len(dnsServers), len(defaultDomains))

	// Run tests
	results := runDNSTests(dnsServers, defaultDomains, time.Duration(*timeoutFlag)*time.Second, *workersFlag)

	// Output results
	if err := outputResults(results, *outputFile, *formatFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error outputting results: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("DNS Check Tool")
	fmt.Println("Usage: dns-check-go [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --list <file>      DNS server list file (IP per line, optional description after space)")
	fmt.Println("  --output <file>    Output file for results (default: stdout)")
	fmt.Printf("  --format <format>  Output format: json, text (default: %s)\n", DefaultFormat)
	fmt.Printf("  --timeout <sec>    Timeout for DNS queries in seconds (default: %d)\n", DefaultTimeout)
	fmt.Printf("  --workers <num>    Number of concurrent workers (default: %d)\n", DefaultWorkerCount)
	fmt.Println("  --help            Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  go run . --list ./dns-servers.txt --output ./results.json")
	fmt.Println("  go run . --output ./results.txt --format text")
	fmt.Println("  go run .  (uses default DNS servers)")
}

func loadDNSServersFromFile(filename string) ([]DNSServer, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var servers []DNSServer
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		ip := parts[0]
		// Validate IP
		if net.ParseIP(ip) == nil {
			fmt.Fprintf(os.Stderr, "Warning: Invalid IP address '%s', skipping\n", ip)
			continue
		}

		server := DNSServer{IP: ip}
		if len(parts) > 1 {
			server.Description = strings.Join(parts[1:], " ")
		}

		servers = append(servers, server)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return servers, nil
}

func runDNSTests(servers []DNSServer, domains []DomainCategory, timeout time.Duration, workers int) TestResults {
	type job struct {
		server DNSServer
		domain DomainCategory
	}

	totalJobs := len(servers) * len(domains)
	jobs := make(chan job, totalJobs)
	results := make(chan TestResult, totalJobs)

	// Progress tracking
	var completedJobs int64
	startTime := time.Now()

	// Start progress bar goroutine
	done := make(chan bool)
	go showProgress(&completedJobs, totalJobs, startTime, done)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				result := testDNS(j.server, j.domain.Domain, timeout)
				result.Category = j.domain.Category
				results <- result
				atomic.AddInt64(&completedJobs, 1)
			}
		}()
	}

	// Send jobs
	go func() {
		defer close(jobs)
		for _, server := range servers {
			for _, domain := range domains {
				jobs <- job{server: server, domain: domain}
			}
		}
	}()

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []TestResult
	for result := range results {
		allResults = append(allResults, result)
	}

	// Stop progress bar
	done <- true
	fmt.Fprintf(os.Stderr, "\n\n")

	// Sort results by server IP then domain
	sort.Slice(allResults, func(i, j int) bool {
		if allResults[i].Server.IP != allResults[j].Server.IP {
			return allResults[i].Server.IP < allResults[j].Server.IP
		}
		return allResults[i].Domain < allResults[j].Domain
	})

	// Calculate summary
	summary := calculateSummary(allResults)

	return TestResults{
		Timestamp: time.Now(),
		Results:   allResults,
		Summary:   summary,
	}
}

func showProgress(completed *int64, total int, startTime time.Time, done chan bool) {
	ticker := time.NewTicker(ProgressUpdateRate)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			current := atomic.LoadInt64(completed)
			elapsed := time.Since(startTime)

			// Calculate progress
			progress := float64(current) / float64(total)
			percentage := progress * 100

			// Calculate ETA
			var eta time.Duration
			if current > 0 {
				avgTimePerJob := elapsed / time.Duration(current)
				remaining := int64(total) - current
				eta = avgTimePerJob * time.Duration(remaining)
			}

			// Create progress bar
			filled := int(progress * float64(ProgressBarWidth))
			bar := strings.Repeat("█", filled) + strings.Repeat("░", ProgressBarWidth-filled)

			// Format time durations
			elapsedStr := formatDuration(elapsed)
			etaStr := formatDuration(eta)

			// Print progress
			fmt.Fprintf(os.Stderr, "\r[%s] %d/%d (%.1f%%) | Elapsed: %s | ETA: %s",
				bar, current, total, percentage, elapsedStr, etaStr)
		}
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

func testDNS(server DNSServer, domain string, timeout time.Duration) TestResult {
	client := &dns.Client{
		Timeout: timeout,
	}

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)

	start := time.Now()
	response, _, err := client.Exchange(msg, net.JoinHostPort(server.IP, "53"))
	responseTime := time.Since(start)

	result := TestResult{
		Server:       server,
		Domain:       domain,
		ResponseTime: responseTime,
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	if response == nil || len(response.Answer) == 0 {
		result.Success = false
		result.Error = "No answer received"
		return result
	}

	// Get the first A record
	for _, answer := range response.Answer {
		if a, ok := answer.(*dns.A); ok {
			result.Success = true
			result.IP = a.A.String()
			break
		}
	}

	if !result.Success {
		result.Error = "No A record found in response"
	}

	return result
}

func calculateSummary(results []TestResult) Summary {
	totalTests := len(results)
	successfulTests := 0
	var totalResponseTime time.Duration

	// Category-based statistics
	categoryStats := make(map[string]CategoryStats)
	categoryResults := make(map[string][]TestResult)

	// Group results by category
	for _, result := range results {
		categoryResults[result.Category] = append(categoryResults[result.Category], result)

		if result.Success {
			successfulTests++
			totalResponseTime += result.ResponseTime
		}
	}

	// Calculate category statistics
	for category, catResults := range categoryResults {
		catTotal := len(catResults)
		catSuccessful := 0

		for _, result := range catResults {
			if result.Success {
				catSuccessful++
			}
		}

		catFailed := catTotal - catSuccessful
		catSuccessRate := float64(catSuccessful) / float64(catTotal) * 100

		categoryStats[category] = CategoryStats{
			TotalTests:      catTotal,
			SuccessfulTests: catSuccessful,
			FailedTests:     catFailed,
			SuccessRate:     catSuccessRate,
		}
	}

	failedTests := totalTests - successfulTests
	successRate := float64(successfulTests) / float64(totalTests) * 100

	var avgResponseTime time.Duration
	if successfulTests > 0 {
		avgResponseTime = totalResponseTime / time.Duration(successfulTests)
	}

	return Summary{
		TotalTests:          totalTests,
		SuccessfulTests:     successfulTests,
		FailedTests:         failedTests,
		SuccessRate:         successRate,
		AverageResponseTime: avgResponseTime,
		CategoryStats:       categoryStats,
	}
}

func outputResults(results TestResults, outputFile, format string) error {
	var output strings.Builder

	switch format {
	case "json":
		jsonData, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		output.Write(jsonData)
	case "text":
		writeTextOutput(&output, results)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if outputFile != "" {
		return os.WriteFile(outputFile, []byte(output.String()), 0644)
	}

	fmt.Print(output.String())
	return nil
}

func writeTextOutput(output *strings.Builder, results TestResults) {
	output.WriteString("DNS Check Results\n")
	output.WriteString("=================\n")
	output.WriteString(fmt.Sprintf("Timestamp: %s\n\n", results.Timestamp.Format("2006-01-02 15:04:05")))

	// Function to write summary
	writeSummary := func() {
		output.WriteString("Summary:\n")
		output.WriteString(fmt.Sprintf("  Total Tests: %d\n", results.Summary.TotalTests))
		output.WriteString(fmt.Sprintf("  Successful: %d\n", results.Summary.SuccessfulTests))
		output.WriteString(fmt.Sprintf("  Failed: %d\n", results.Summary.FailedTests))
		output.WriteString(fmt.Sprintf("  Overall Success Rate: %.2f%%\n", results.Summary.SuccessRate))
		output.WriteString(fmt.Sprintf("  Average Response Time: %v\n", results.Summary.AverageResponseTime))

		// Category-based summary
		output.WriteString("\n  Category Success Rates:\n")
		for _, category := range CategoryOrder {
			if stats, exists := results.Summary.CategoryStats[category]; exists {
				output.WriteString(fmt.Sprintf("    %-12s: %.2f%% (%d/%d)\n",
					category, stats.SuccessRate, stats.SuccessfulTests, stats.TotalTests))
			}
		}
		output.WriteString("\n")
	}

	// Summary at the beginning
	writeSummary()

	// Group results by server
	serverResults := make(map[string][]TestResult)
	for _, result := range results.Results {
		key := result.Server.IP
		if result.Server.Description != "" {
			key += " (" + result.Server.Description + ")"
		}
		serverResults[key] = append(serverResults[key], result)
	}

	// Sort servers
	var servers []string
	for server := range serverResults {
		servers = append(servers, server)
	}
	sort.Strings(servers)

	// Output results by server
	output.WriteString("Detailed Results:\n")
	output.WriteString("-----------------\n")

	for _, server := range servers {
		output.WriteString(fmt.Sprintf("\nDNS Server: %s\n", server))

		// Group by category
		categoryResults := make(map[string][]TestResult)
		for _, result := range serverResults[server] {
			categoryResults[result.Category] = append(categoryResults[result.Category], result)
		}

		totalSuccessful := 0
		for _, category := range CategoryOrder {
			if results, exists := categoryResults[category]; exists {
				output.WriteString(fmt.Sprintf("  %s:\n", category))

				categorySuccessful := 0
				for _, result := range results {
					status := "FAIL"
					details := result.Error
					if result.Success {
						status = "OK"
						details = result.IP
						categorySuccessful++
						totalSuccessful++
					}

					output.WriteString(fmt.Sprintf("    %-22s [%4s] %8v %s\n",
						result.Domain, status, result.ResponseTime.Truncate(time.Millisecond), details))
				}

				categoryRate := float64(categorySuccessful) / float64(len(results)) * 100
				output.WriteString(fmt.Sprintf("    %s Success Rate: %.2f%% (%d/%d)\n\n",
					category, categoryRate, categorySuccessful, len(results)))
			}
		}

		overallRate := float64(totalSuccessful) / float64(len(serverResults[server])) * 100
		output.WriteString(fmt.Sprintf("  Overall Success Rate: %.2f%% (%d/%d)\n",
			overallRate, totalSuccessful, len(serverResults[server])))
	}

	// Summary at the end
	output.WriteString("\n")
	output.WriteString("=================\n")
	writeSummary()
}
