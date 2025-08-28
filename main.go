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
	{"discord.com", CategoryGeneral},
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
	{"roblox.com", CategoryOther},

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

/*
Source: DNSJumper Application
*/
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

/*
Source: https://dnsmid.com/turkey/
var defaultDNSServers = []DNSServer{
	{"94.73.166.124", "AS34619 - CIZGI TELEKOMUNIKASYON ANONIM SIRKETI (Turkey, Şişli)"},
	{"194.5.236.109", "AS209828 - Genc BT Bilisim Teknolojileri Limited Sirketi (Türkiye, Istanbul)"},
	{"212.68.34.124", "AS212219 - HOSTING DUNYAM (Turkey, Istanbul)"},
	{"77.92.138.166", "AS42910 - PremierDC Veri Merkezi Anonim Sirketi (Turkey, Eyüpsultan)"},
	{"195.21.58.113", "AS8928 - GTT Communications Inc. (Turkey, Istanbul)"},
	{"92.45.47.114", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"95.9.37.245", "AS9121 - Turk Telekomunikasyon Anonim Sirketi (Turkey, Kadıköy)"},
	{"195.244.44.44", "AS43391 - Netdirekt A.S. (Turkey, Konak)"},
	{"90.158.111.168", "AS9021 - Is Net Elektonik Bilgi Uretim Dagitim Ticaret ve Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"176.53.10.136", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey, Istanbul)"},
	{"88.248.51.121", "AS47331 - TTNet A.S. (Turkey, Antalya)"},
	{"78.135.102.237", "AS8685 - Doruk Iletisim ve Otomasyon Sanayi ve Ticaret A.S. (Turkey, Sisli)"},
	{"89.19.14.82", "AS34619 - CIZGI TELEKOMUNIKASYON ANONIM SIRKETI (Turkey, Sisli)"},
	{"213.153.223.21", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"213.194.123.26", "AS15924 - Vodafone Net Iletisim Hizmetler AS (Turkey, Istanbul)"},
	{"213.74.195.52", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Umraniye)"},
	{"176.88.18.85", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Umraniye)"},
	{"208.67.222.222", "AS36692 - Cisco OpenDNS, LLC (United States, Wright City)"},
	{"195.46.39.39", "AS57926 - SafeDNS, Inc. (Germany, Frankfurt)"},
	{"8.8.8.8", "AS15169 - Google LLC (United States, Ashburn)"},
}*/

/*
Source: https://public-dns.info/nameserver/tr.html

var defaultDNSServers = []DNSServer{
	{"95.9.194.13", "AS47331 - Turk Telekom (Turkey, Konyaalti)"},
	{"92.45.59.195", "AS34984 - Tellcom Iletisim Hizmetleri A.s. (Turkey, Istanbul)"},
	{"93.184.144.5", "AS47288 - FIXNET Telekomunikasyon Limited Sirketi (Turkey, Edirne)"},
	{"94.54.47.200", "AS47524 - Turksat Uydu Haberlesme ve Kablo TV Isletme A.S. (Turkey, Ankara)"},
	{"176.53.92.22", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"185.250.193.165", "AS201079 - AKA Bilisim Yazilim Arge Ins. Taah. San. Tic. A.S. (Turkey)"},
	{"213.14.10.165", "AS34984 - Tellcom Iletisim Hizmetleri A.s. (Turkey, Istanbul)"},
	{"62.248.9.91", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"176.235.200.91", "AS34984 - Tellcom Iletisim Hizmetleri A.s. (Turkey)"},
	{"212.98.235.50", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Istanbul)"},
	{"176.236.223.149", "AS34984 - Tellcom Iletisim Hizmetleri A.s. (Turkey)"},
	{"176.236.223.156", "AS34984 - Tellcom Iletisim Hizmetleri A.s. (Turkey)"},
	{"185.200.36.182", "AS202561 - High Speed Telekomunikasyon ve Hab. Hiz. Ltd. Sti. (Turkey)"},
	{"31.7.36.36", "AS57152 - Teknet Yazlim Ve Bilgisayar Teknolojileri (Turkey, Antalya)"},
	{"31.7.37.37", "AS57152 - Teknet Yazlim Ve Bilgisayar Teknolojileri (Turkey, Antalya)"},
	{"2.59.119.25", "AS212219 - Talha Bogaz (Turkey)"},
	{"5.25.56.138", "AS16135 - Turkcell Iletisim Hizmetleri A.s. (Turkey, Istanbul)"},
	{"5.25.82.30", "AS16135 - Turkcell Iletisim Hizmetleri A.s. (Turkey, Istanbul)"},
	{"5.25.98.224", "AS16135 - Turkcell Iletisim Hizmetleri A.s. (Turkey, Istanbul)"},
	{"5.25.116.109", "AS16135 - Turkcell Iletisim Hizmetleri A.s. (Turkey, Istanbul)"},
	{"5.25.113.223", "AS16135 - Turkcell Iletisim Hizmetleri A.s. (Turkey, Istanbul)"},
	{"45.195.77.74", "AS43260 - Dgn Teknoloji A.s. (Turkey)"},
	{"193.192.113.130", "AS12735 - TurkNet Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"78.135.69.51", "AS205953 - NAZNET Bilisim ve Telekomunikasyon Elektronik Haberlesme Hiz. ith. ihr. San. ve Tic. Ltd. s (Turkey, Nazilli)"},
	{"45.195.77.57", "AS43260 - Dgn Teknoloji A.s. (Turkey)"},
	{"176.235.165.91", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"176.53.85.154", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"88.249.41.163", "AS9121 - Turk Telekom (Turkey)"},
	{"212.125.13.61", "AS12735 - TurkNet Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"78.135.81.205", "AS207326 - HostLAB Bilisim Teknolojileri A.S. (Turkey, Istanbul)"},
	{"31.145.58.190", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Tekirdağ)"},
	{"195.142.119.220", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"212.68.34.124", "AS212219 - Talha Bogaz (Turkey)"},
	{"212.68.40.6", "AS42910 - PremierDC Veri Merkezi Anonim Sirketi (Turkey)"},
	{"149.0.16.217", "AS8386 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Bursa)"},
	{"213.248.179.191", "AS8386 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey)"},
	{"109.228.208.152", "AS34296 - Millenicom Telekomunikasyon Hizmetleri Anonim Sirketi (Turkey, Antalya)"},
	{"77.73.216.5", "AS42716 - Assan Bilisim A.S. (Turkey, Istanbul)"},
	{"82.222.48.42", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"46.221.5.200", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Istanbul)"},
	{"91.93.153.68", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey)"},
	{"213.153.223.21", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"78.135.107.209", "AS211859 - Ozkula Internet Hizmetleri Tic. LTD. STI. (Turkey)"},
	{"31.206.250.130", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Denizli)"},
	{"178.211.56.71", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"185.67.205.193", "AS59886 - Layer Sistem tic. ltd. sti. (Turkey)"},
	{"78.189.141.206", "AS9121 - Turk Telekom (Turkey, Bursa)"},
	{"94.78.85.253", "AS44558 - Netonline Bilisim Sirketi LTD (Turkey)"},
	{"85.98.211.107", "AS9121 - Turk Telekom (Turkey, Denizli)"},
	{"31.145.56.14", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Istanbul)"},
	{"85.29.51.9", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"45.11.96.36", "AS48678 - PENTECH BILISIM TEKNOLOJILERI SANAYI VE TICARET LIMITED SIRKETi (Turkey)"},
	{"176.88.41.253", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"88.250.207.65", "AS9121 - Turk Telekom (Turkey, Izmir)"},
	{"78.186.181.121", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"31.145.110.132", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Istanbul)"},
	{"213.248.134.130", "AS8386 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Yukarikaraman)"},
	{"88.249.67.177", "AS9121 - Turk Telekom (Turkey, Akçaabat)"},
	{"194.145.138.11", "AS204457 - Atlantis Telekomunikasyon Bilisim Hizmetleri San. Tic. Ltd (Turkey, Istanbul)"},
	{"46.221.14.44", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Bigadic)"},
	{"46.196.212.8", "AS47524 - Turksat Uydu Haberlesme ve Kablo TV Isletme A.S. (Turkey, Gaziantep)"},
	{"31.210.79.250", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey, Istanbul)"},
	{"194.5.236.248", "AS209828 - Genc BT Bilisim Teknolojileri Limited Sirketi (Turkey)"},
	{"85.98.93.171", "AS9121 - Turk Telekom (Turkey, Didim)"},
	{"95.9.85.219", "AS9121 - Turk Telekom (Turkey, Kayseri)"},
	{"204.157.133.81", "AS44547 - Netundweb Telekomunikasyon Ticaret Limited Sirketi (Turkey)"},
	{"88.247.99.66", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"185.250.240.179", "AS211804 - Sistemdc webhosting and server services (Turkey)"},
	{"213.153.224.29", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"176.235.135.204", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey)"},
	{"91.93.139.159", "AS43352 - Teletek Bulut Bilisim ve Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"204.157.133.250", "AS44547 - Netundweb Telekomunikasyon Ticaret Limited Sirketi (Turkey)"},
	{"24.133.181.111", "AS47524 - Turksat Uydu Haberlesme ve Kablo TV Isletme A.S. (Turkey, Ankara)"},
	{"31.192.208.20", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"176.88.166.103", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Burdur)"},
	{"204.157.133.30", "AS44547 - Netundweb Telekomunikasyon Ticaret Limited Sirketi (Turkey)"},
	{"213.153.224.28", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"212.98.231.69", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey)"},
	{"37.1.145.34", "AS50941 - Vargonen Teknoloji ve Bilisim Sanayi Ticaret Anonim Sirketi (Turkey)"},
	{"37.1.145.102", "AS50941 - Vargonen Teknoloji ve Bilisim Sanayi Ticaret Anonim Sirketi (Turkey)"},
	{"31.206.52.78", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Kosekoy)"},
	{"82.222.57.87", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"188.132.234.170", "AS42910 - PremierDC Veri Merkezi Anonim Sirketi (Turkey, Istanbul)"},
	{"212.98.224.69", "AS48678 - PENTECH BILISIM TEKNOLOJILERI SANAYI VE TICARET LIMITED SIRKETi (Turkey)"},
	{"85.96.196.109", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"45.156.31.240", "AS204457 - Atlantis Telekomunikasyon Bilisim Hizmetleri San. Tic. Ltd (Turkey, Istanbul)"},
	{"195.142.127.83", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Konya)"},
	{"185.92.2.100", "AS202536 - Isim Kayit Bilisim (Turkey, Kosekoy)"},
	{"213.153.229.193", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"195.21.58.113", "AS8928 - GTT Communications Inc. (Turkey)"},
	{"38.242.150.124", "AS51167 - Contabo GmbH (Turkey, Diyarbakır)"},
	{"78.186.250.194", "AS9121 - Turk Telekom (Turkey, Bursa)"},
	{"89.19.8.120", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"212.98.224.174", "AS48678 - PENTECH BILISIM TEKNOLOJILERI SANAYI VE TICARET LIMITED SIRKETi (Turkey)"},
	{"176.236.37.163", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"77.79.101.131", "AS39582 - Grid Telekomunikasyon Hizmetleri AS (Turkey)"},
	{"95.9.250.227", "AS47331 - Turk Telekom (Turkey, Adana)"},
	{"37.75.10.106", "AS199484 - SAGLAYICI Teknoloji Bilisim Yayincilik Hiz. Ticaret Ltd. Sti. (Turkey, Istanbul)"},
	{"91.151.83.147", "AS60707 - Kapteyan Bilisim Teknolojileri San. ve Tic. A.S. (Turkey)"},
	{"79.98.134.211", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"80.253.246.63", "AS212219 - Talha Bogaz (Turkey)"},
	{"195.33.236.164", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Izmir)"},
	{"188.3.122.46", "AS8386 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Istanbul)"},
	{"78.188.115.140", "AS9121 - Turk Telekom (Turkey, Fatih)"},
	{"88.247.23.246", "AS9121 - Turk Telekom (Turkey, Antalya)"},
	{"194.62.40.20", "AS42724 - Talido Bilisim Teknolojileri A.S (Turkey, Istanbul)"},
	{"88.225.232.205", "AS9121 - Turk Telekom (Turkey, Antalya)"},
	{"78.187.76.85", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"95.173.168.180", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"78.189.170.100", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"213.74.223.74", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"91.93.172.170", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"88.247.215.148", "AS9121 - Turk Telekom (Turkey, Germencik)"},
	{"88.238.138.10", "AS9121 - Turk Telekom (Turkey, Ankara)"},
	{"81.8.30.102", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Kadirli)"},
	{"78.168.240.115", "AS47331 - Turk Telekom (Turkey, Adana)"},
	{"195.46.129.94", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Alanya)"},
	{"81.214.127.23", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"88.247.26.195", "AS47331 - Turk Telekom (Turkey, Niğde)"},
	{"92.45.47.28", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"88.225.219.169", "AS9121 - Turk Telekom (Turkey, Konya)"},
	{"185.248.14.200", "AS204457 - Atlantis Telekomunikasyon Bilisim Hizmetleri San. Tic. Ltd (Turkey, Esenyurt)"},
	{"95.9.226.53", "AS9121 - Turk Telekom (Turkey, Antalya)"},
	{"185.153.222.232", "AS49126 - IHS Kurumsal Teknoloji Hizmetleri A.S (Turkey)"},
	{"213.142.148.40", "AS212219 - Talha Bogaz (Turkey)"},
	{"38.10.71.214", "AS208972 - Gibirnet Iletisim Hizmetleri Sanayi Ve Ticaret Limited Sirketi (Turkey, Sanliurfa)"},
	{"81.214.36.153", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"85.105.252.93", "AS47331 - Turk Telekom (Turkey, Ankara)"},
	{"94.102.13.152", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"91.151.88.109", "AS212219 - Talha Bogaz (Turkey)"},
	{"78.186.173.215", "AS9121 - Turk Telekom (Turkey, Mugla)"},
	{"78.189.16.71", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"94.73.156.76", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"213.74.249.98", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Çanakkale)"},
	{"94.73.156.78", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"212.58.26.33", "AS8685 - Doruk Iletisim ve Otomasyon Sanayi ve Ticaret A.S. (Turkey)"},
	{"185.208.102.147", "AS202561 - High Speed Telekomunikasyon ve Hab. Hiz. Ltd. Sti. (Turkey, Kilis)"},
	{"81.214.12.150", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"81.213.150.85", "AS47331 - Turk Telekom (Turkey, Nilufer)"},
	{"78.188.63.155", "AS47331 - Turk Telekom (Turkey, Elâzığ)"},
	{"94.124.73.153", "AS208095 - Internetten Teknoloji Bil.san.tic.ltd.sti. (Turkey)"},
	{"78.189.46.123", "AS9121 - Turk Telekom (Turkey, Avcilar)"},
	{"31.210.52.168", "AS49334 - Sh Online Iletisim Anonim Sirketi (Turkey)"},
	{"82.222.60.46", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"95.8.200.28", "AS47331 - Turk Telekom (Turkey, Istanbul)"},
	{"85.108.206.75", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"185.85.207.35", "AS201079 - AKA Bilisim Yazilim Arge Ins. Taah. San. Tic. A.S. (Turkey)"},
	{"185.140.125.220", "AS57152 - Teknet Yazlim Ve Bilgisayar Teknolojileri (Turkey)"},
	{"188.132.221.60", "AS202561 - High Speed Telekomunikasyon ve Hab. Hiz. Ltd. Sti. (Turkey, Antakya)"},
	{"185.15.198.12", "AS201520 - Dedicated Telekomunikasyon Teknoloji Hiz. Tic. San. LTD. STI. (Turkey)"},
	{"78.188.246.235", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"95.142.132.78", "AS49840 - Enson Net Ltd (Turkey)"},
	{"88.249.68.14", "AS47331 - Turk Telekom (Turkey, Niğde)"},
	{"176.236.77.102", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"89.43.31.10", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"89.252.167.61", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"77.75.35.138", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"78.189.180.162", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"95.9.95.194", "AS9121 - Turk Telekom (Turkey, Ankara)"},
	{"193.192.113.146", "AS12735 - TurkNet Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"82.222.49.18", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"88.227.153.194", "AS47331 - Turk Telekom (Turkey, Cankaya)"},
	{"95.9.241.172", "AS47331 - Turk Telekom (Turkey, Adana)"},
	{"88.234.22.129", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"194.124.36.207", "AS202561 - High Speed Telekomunikasyon ve Hab. Hiz. Ltd. Sti. (Turkey, Antalya)"},
	{"85.105.18.139", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"95.173.162.124", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"78.187.231.245", "AS47331 - Turk Telekom (Turkey, Ankara)"},
	{"85.106.31.119", "AS9121 - Turk Telekom (Turkey, Diyarbakır)"},
	{"213.14.66.51", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Izmir)"},
	{"185.169.183.54", "AS206119 - Veganet Teknolojileri ve Hizmetleri LTD STI (Turkey, Istanbul)"},
	{"95.0.226.132", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"94.102.6.240", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"94.102.7.90", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"185.8.12.24", "AS61345 - Flytom Networks Ltd (Turkey, Mersin)"},
	{"89.19.8.118", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"81.213.79.71", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"95.173.162.118", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"80.93.216.18", "AS20649 - FS Veri Merkezi Internet Teknolojileri Limited Sirketi (Turkey)"},
	{"94.101.87.175", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"85.99.244.122", "AS9121 - Turk Telekom (Turkey, Fethiye)"},
	{"185.208.102.49", "AS202561 - High Speed Telekomunikasyon ve Hab. Hiz. Ltd. Sti. (Turkey, Kilis)"},
	{"94.124.73.151", "AS208095 - Internetten Teknoloji Bil.san.tic.ltd.sti. (Turkey)"},
	{"78.188.58.228", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"78.189.229.223", "AS9121 - Turk Telekom (Turkey, Izmir)"},
	{"77.92.133.5", "AS42910 - PremierDC Veri Merkezi Anonim Sirketi (Turkey)"},
	{"88.249.58.86", "AS47331 - Turk Telekom (Turkey, Kayseri)"},
	{"176.88.10.83", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Sisli)"},
	{"77.75.37.178", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"194.124.36.201", "AS202561 - High Speed Telekomunikasyon ve Hab. Hiz. Ltd. Sti. (Turkey, Antalya)"},
	{"77.223.142.102", "AS43391 - Netdirekt A.S. (Turkey)"},
	{"78.186.206.121", "AS9121 - Turk Telekom (Turkey, Izmir)"},
	{"176.53.35.127", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"95.9.190.34", "AS9121 - Turk Telekom (Turkey, Konya)"},
	{"92.45.47.114", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"176.88.18.87", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Ankara)"},
	{"77.223.128.221", "AS43391 - Netdirekt A.S. (Turkey)"},
	{"213.194.123.26", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Antalya)"},
	{"78.111.106.91", "AS20649 - FS Veri Merkezi Internet Teknolojileri Limited Sirketi (Turkey)"},
	{"77.75.35.139", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"78.182.254.16", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"89.19.16.37", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"88.227.51.145", "AS9121 - Turk Telekom (Turkey, Basaksehir)"},
	{"85.153.132.196", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"85.95.240.30", "AS206991 - Iksir Internet Hizmetleri A.S. (Turkey)"},
	{"78.188.115.188", "AS9121 - Turk Telekom (Turkey, Fatih)"},
	{"193.192.121.230", "AS12735 - TurkNet Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"212.57.11.185", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Mugla)"},
	{"78.135.102.237", "AS8685 - Doruk Iletisim ve Otomasyon Sanayi ve Ticaret A.S. (Turkey)"},
	{"77.92.138.166", "AS42910 - PremierDC Veri Merkezi Anonim Sirketi (Turkey)"},
	{"88.250.243.205", "AS9121 - Turk Telekom (Turkey, Mugla)"},
	{"90.158.111.168", "AS9021 - Is Net Elektonik Bilgi Uretim Dagitim Ticaret ve Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"38.242.147.147", "AS51167 - Contabo GmbH (Turkey, Diyarbakır)"},
	{"78.188.181.64", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"78.189.127.251", "AS47331 - Turk Telekom (Turkey, Adana)"},
	{"93.94.252.171", "AS47123 - TI Sparkle Turkey Telekomunukasyon A.S (Turkey)"},
	{"88.236.253.84", "AS47331 - Turk Telekom (Turkey, Istanbul)"},
	{"88.242.0.188", "AS47331 - Turk Telekom (Turkey, Istanbul)"},
	{"84.44.14.35", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Istanbul)"},
	{"78.135.85.37", "AS212219 - Talha Bogaz (Turkey)"},
	{"194.124.36.68", "AS202561 - High Speed Telekomunikasyon ve Hab. Hiz. Ltd. Sti. (Turkey, Antalya)"},
	{"212.68.34.235", "AS212219 - Talha Bogaz (Turkey)"},
	{"37.148.213.124", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"94.73.160.131", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"185.33.232.4", "AS51557 - Isimtescil Bilisim A.S. (Turkey)"},
	{"95.9.108.212", "AS9121 - Turk Telekom (Turkey, Osmaniye)"},
	{"195.46.151.9", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Istanbul)"},
	{"93.184.152.105", "AS47288 - FIXNET Telekomunikasyon Limited Sirketi (Turkey, Istanbul)"},
	{"85.95.242.71", "AS206991 - Iksir Internet Hizmetleri A.S. (Turkey)"},
	{"85.99.232.65", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"84.51.47.66", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Osmaniye)"},
	{"95.9.130.102", "AS9121 - Turk Telekom (Turkey, Kahramanmaraş)"},
	{"77.92.133.11", "AS42910 - PremierDC Veri Merkezi Anonim Sirketi (Turkey)"},
	{"176.235.221.96", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey)"},
	{"78.188.215.107", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"185.22.187.150", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"31.210.78.210", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey, Istanbul)"},
	{"195.142.153.122", "AS199484 - SAGLAYICI Teknoloji Bilisim Yayincilik Hiz. Ticaret Ltd. Sti. (Turkey, Istanbul)"},
	{"77.223.128.218", "AS43391 - Netdirekt A.S. (Turkey)"},
	{"85.95.238.173", "AS206991 - Iksir Internet Hizmetleri A.S. (Turkey)"},
	{"88.250.66.143", "AS9121 - Turk Telekom (Turkey, Adapazarı)"},
	{"78.186.131.161", "AS9121 - Turk Telekom (Turkey, Kadıköy)"},
	{"45.156.31.140", "AS204457 - Atlantis Telekomunikasyon Bilisim Hizmetleri San. Tic. Ltd (Turkey, Istanbul)"},
	{"94.138.223.150", "AS49126 - IHS Kurumsal Teknoloji Hizmetleri A.S (Turkey)"},
	{"94.102.6.241", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"213.14.66.54", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Izmir)"},
	{"78.186.134.59", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"77.90.131.59", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Kadıköy)"},
	{"91.191.170.4", "AS43391 - Netdirekt A.S. (Turkey)"},
	{"81.213.148.253", "AS47331 - Turk Telekom (Turkey, Nilufer)"},
	{"89.19.22.46", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"88.249.166.158", "AS9121 - Turk Telekom (Turkey, Çanakkale)"},
	{"78.184.250.152", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"176.236.139.87", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Mersin)"},
	{"78.179.111.225", "AS47331 - Turk Telekom (Turkey, Istanbul)"},
	{"176.88.112.53", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Mugla)"},
	{"185.117.123.41", "AS213145 - Fibim Fibernet Gsm Sanayi Ve Ticaret Anonim Sirketi (Turkey, Alanya)"},
	{"78.186.36.168", "AS47331 - Turk Telekom (Turkey, Cankaya)"},
	{"95.173.162.126", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"93.184.152.43", "AS47288 - FIXNET Telekomunikasyon Limited Sirketi (Turkey, Istanbul)"},
	{"193.36.63.184", "AS201086 - ServerPlusInternet Sunucu Hizmetleri (Turkey, Bursa)"},
	{"78.186.159.117", "AS47331 - Turk Telekom (Turkey, Etimesgut)"},
	{"85.100.138.3", "AS47331 - Turk Telekom (Turkey, Adana)"},
	{"78.188.10.14", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"185.226.160.76", "AS205192 - Omur Bilisim Teknolojileri (Turkey, Ankara)"},
	{"93.184.152.109", "AS47288 - FIXNET Telekomunikasyon Limited Sirketi (Turkey, Istanbul)"},
	{"92.45.47.27", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"89.252.173.104", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"176.88.18.88", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Ankara)"},
	{"195.87.69.30", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Bursa)"},
	{"85.98.40.40", "AS47331 - Turk Telekom (Turkey)"},
	{"78.189.111.211", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"94.124.73.139", "AS208095 - Internetten Teknoloji Bil.san.tic.ltd.sti. (Turkey)"},
	{"91.93.64.53", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey)"},
	{"88.250.238.6", "AS9121 - Turk Telekom (Turkey, Menemen)"},
	{"88.228.16.148", "AS9121 - Turk Telekom (Turkey, Ankara)"},
	{"78.189.47.69", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"78.111.106.90", "AS20649 - FS Veri Merkezi Internet Teknolojileri Limited Sirketi (Turkey)"},
	{"93.177.103.194", "AS207326 - HostLAB Bilisim Teknolojileri A.S. (Turkey, Istanbul)"},
	{"84.44.14.37", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Istanbul)"},
	{"78.189.237.37", "AS9121 - Turk Telekom (Turkey, Akhisar)"},
	{"176.88.181.98", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Adana)"},
	{"84.44.9.10", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Istanbul)"},
	{"93.184.152.108", "AS47288 - FIXNET Telekomunikasyon Limited Sirketi (Turkey, Istanbul)"},
	{"91.93.131.69", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Eyuepsultan)"},
	{"78.186.49.28", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"94.102.8.154", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"212.125.11.12", "AS12735 - TurkNet Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"88.248.55.71", "AS9121 - Turk Telekom (Turkey, Mugla)"},
	{"88.248.56.216", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"89.19.7.213", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"88.247.206.159", "AS9121 - Turk Telekom (Turkey, Izmir)"},
	{"82.222.152.165", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey)"},
	{"195.46.154.156", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Istanbul)"},
	{"91.102.162.194", "AS41801 - Datafon Teknoloji San.Tic.Ltd.Sti. (Turkey)"},
	{"217.65.177.83", "AS34296 - Millenicom Telekomunikasyon Hizmetleri Anonim Sirketi (Turkey, Kayseri)"},
	{"92.45.25.227", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Akhisar)"},
	{"93.184.146.30", "AS47288 - FIXNET Telekomunikasyon Limited Sirketi (Turkey, Kırklareli)"},
	{"78.186.194.187", "AS9121 - Turk Telekom (Turkey, Izmir)"},
	{"188.132.221.58", "AS202561 - High Speed Telekomunikasyon ve Hab. Hiz. Ltd. Sti. (Turkey, Antakya)"},
	{"213.74.195.52", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Cankaya)"},
	{"78.188.205.173", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"92.45.200.125", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"185.233.247.205", "AS206119 - Veganet Teknolojileri ve Hizmetleri LTD STI (Turkey)"},
	{"194.146.50.241", "AS200456 - Verigom Telekomunikasyon Limited Sirketi (Turkey)"},
	{"95.173.162.111", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"85.105.94.129", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"81.215.214.136", "AS9121 - Turk Telekom (Turkey, Tokat Province)"},
	{"213.155.107.188", "AS8685 - Doruk Iletisim ve Otomasyon Sanayi ve Ticaret A.S. (Turkey)"},
	{"94.54.12.73", "AS47524 - Turksat Uydu Haberlesme ve Kablo TV Isletme A.S. (Turkey, Mersin)"},
	{"88.247.165.243", "AS9121 - Turk Telekom (Turkey, Izmir)"},
	{"88.248.195.254", "AS9121 - Turk Telekom (Turkey, Muratpasa)"},
	{"176.53.35.120", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"88.248.104.242", "AS9121 - Turk Telekom (Turkey, Edremit)"},
	{"94.73.154.204", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"91.93.131.70", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Eyuepsultan)"},
	{"85.106.20.174", "AS9121 - Turk Telekom (Turkey, Cekmekoey)"},
	{"94.54.88.208", "AS47524 - Turksat Uydu Haberlesme ve Kablo TV Isletme A.S. (Turkey, Ankara)"},
	{"81.214.73.194", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"94.102.76.212", "AS8685 - Doruk Iletisim ve Otomasyon Sanayi ve Ticaret A.S. (Turkey)"},
	{"85.102.255.171", "AS47331 - Turk Telekom (Turkey, Istanbul)"},
	{"81.214.69.83", "AS9121 - Turk Telekom (Turkey, Samsun)"},
	{"93.184.146.28", "AS47288 - FIXNET Telekomunikasyon Limited Sirketi (Turkey, Kırklareli)"},
	{"93.113.63.27", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"85.97.199.245", "AS9121 - Turk Telekom (Turkey, Denizli)"},
	{"94.101.87.231", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"78.189.29.186", "AS9121 - Turk Telekom (Turkey, Kartal)"},
	{"82.150.94.174", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Istanbul)"},
	{"195.46.129.74", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Alanya)"},
	{"88.249.51.252", "AS9121 - Turk Telekom (Turkey, Alanya)"},
	{"78.186.120.157", "AS47331 - Turk Telekom (Turkey, Adalar)"},
	{"185.81.153.147", "AS202505 - Netbudur Telekomunikasyon Limited Sirketi (Turkey)"},
	{"88.245.96.166", "AS9121 - Turk Telekom (Turkey, Kütahya)"},
	{"78.189.137.251", "AS47331 - Turk Telekom (Turkey, Istanbul)"},
	{"88.248.51.121", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"85.99.234.230", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"88.250.55.67", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"81.214.254.111", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"92.45.47.30", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"95.9.81.100", "AS47331 - Turk Telekom (Turkey, Kayseri)"},
	{"85.95.244.88", "AS206991 - Iksir Internet Hizmetleri A.S. (Turkey)"},
	{"31.169.79.37", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"92.44.191.15", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Nilufer)"},
	{"94.138.207.90", "AS49126 - IHS Kurumsal Teknoloji Hizmetleri A.S (Turkey)"},
	{"85.159.71.156", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"77.245.158.121", "AS42868 - Niobe Bilisim Teknolojileri Yazilim San. Tic. Ltd. Sti. (Turkey)"},
	{"91.241.49.28", "AS209828 - Genc BT Bilisim Teknolojileri Limited Sirketi (Turkey)"},
	{"91.191.173.202", "AS43391 - Netdirekt A.S. (Turkey)"},
	{"88.250.224.28", "AS9121 - Turk Telekom (Turkey, Bursa)"},
	{"78.188.7.231", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"188.132.193.152", "AS201233 - Yonca Duran (Turkey)"},
	{"94.102.6.239", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"81.214.141.75", "AS47331 - Turk Telekom (Turkey, Istanbul)"},
	{"176.53.10.136", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey, Istanbul)"},
	{"82.222.48.19", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"77.92.155.196", "AS42910 - PremierDC Veri Merkezi Anonim Sirketi (Turkey)"},
	{"213.238.172.225", "AS60707 - Kapteyan Bilisim Teknolojileri San. ve Tic. A.S. (Turkey, Istanbul)"},
	{"85.95.242.120", "AS206991 - Iksir Internet Hizmetleri A.S. (Turkey)"},
	{"91.93.132.36", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"94.101.82.233", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"91.151.94.246", "AS211560 - Muhammet Ugur Ozturk (Turkey)"},
	{"94.102.2.244", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"213.14.11.162", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Yildirim)"},
	{"31.145.137.156", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Bursa)"},
	{"31.210.69.163", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey, Istanbul)"},
	{"78.188.22.60", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"92.44.44.184", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey)"},
	{"212.57.29.100", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"88.247.151.174", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"185.48.180.218", "AS49126 - IHS Kurumsal Teknoloji Hizmetleri A.S (Turkey)"},
	{"85.95.238.113", "AS206991 - Iksir Internet Hizmetleri A.S. (Turkey)"},
	{"95.9.37.245", "AS9121 - Turk Telekom (Turkey, Karatay)"},
	{"91.93.58.175", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Adana)"},
	{"212.98.194.132", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Seyhan)"},
	{"95.6.86.31", "AS9121 - Turk Telekom (Turkey, Soeke)"},
	{"95.173.162.85", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"85.102.10.64", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"81.8.30.98", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Kadirli)"},
	{"94.138.206.14", "AS49126 - IHS Kurumsal Teknoloji Hizmetleri A.S (Turkey)"},
	{"185.200.36.93", "AS202561 - High Speed Telekomunikasyon ve Hab. Hiz. Ltd. Sti. (Turkey, Reyhanli)"},
	{"85.95.238.114", "AS206991 - Iksir Internet Hizmetleri A.S. (Turkey)"},
	{"78.186.138.124", "AS9121 - Turk Telekom (Turkey, Izmir)"},
	{"94.73.166.124", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"194.145.138.232", "AS204457 - Atlantis Telekomunikasyon Bilisim Hizmetleri San. Tic. Ltd (Turkey, Istanbul)"},
	{"81.8.106.42", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Talas)"},
	{"78.188.42.166", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"78.188.37.78", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"78.186.60.77", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"88.250.245.64", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"81.213.149.81", "AS47331 - Turk Telekom (Turkey, Kayapinar)"},
	{"85.95.242.156", "AS206991 - Iksir Internet Hizmetleri A.S. (Turkey)"},
	{"78.189.139.221", "AS47331 - Turk Telekom (Turkey, Ankara)"},
	{"93.184.146.27", "AS47288 - FIXNET Telekomunikasyon Limited Sirketi (Turkey, Kırklareli)"},
	{"85.97.195.137", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"185.85.204.177", "AS201079 - AKA Bilisim Yazilim Arge Ins. Taah. San. Tic. A.S. (Turkey)"},
	{"85.99.242.115", "AS9121 - Turk Telekom (Turkey, Izmir)"},
	{"91.93.153.74", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"85.105.171.92", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"78.111.106.93", "AS20649 - FS Veri Merkezi Internet Teknolojileri Limited Sirketi (Turkey)"},
	{"176.53.35.113", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"94.138.223.151", "AS49126 - IHS Kurumsal Teknoloji Hizmetleri A.S (Turkey)"},
	{"88.248.99.53", "AS9121 - Turk Telekom (Turkey, Balıkesir)"},
	{"78.186.25.166", "AS47331 - Turk Telekom (Turkey)"},
	{"95.9.233.135", "AS47331 - Turk Telekom (Turkey)"},
	{"77.79.92.164", "AS39582 - Grid Telekomunikasyon Hizmetleri AS (Turkey, Zonguldak)"},
	{"88.247.20.13", "AS9121 - Turk Telekom (Turkey, Karatay)"},
	{"94.73.156.77", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"81.215.63.47", "AS47331 - Turk Telekom (Turkey, Van)"},
	{"78.186.129.210", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"93.177.103.34", "AS207326 - HostLAB Bilisim Teknolojileri A.S. (Turkey, Istanbul)"},
	{"84.51.15.20", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Konya)"},
	{"188.132.205.174", "AS42910 - PremierDC Veri Merkezi Anonim Sirketi (Turkey)"},
	{"93.177.103.23", "AS207326 - HostLAB Bilisim Teknolojileri A.S. (Turkey, Istanbul)"},
	{"95.6.70.227", "AS9121 - Turk Telekom (Turkey, Bagcilar)"},
	{"85.105.122.250", "AS47331 - Turk Telekom (Turkey, Ankara)"},
	{"90.158.200.15", "AS9021 - Is Net Elektonik Bilgi Uretim Dagitim Ticaret ve Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"78.135.113.203", "AS42910 - PremierDC Veri Merkezi Anonim Sirketi (Turkey)"},
	{"81.22.109.110", "AS48737 - Dorabase Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"85.95.239.215", "AS206991 - Iksir Internet Hizmetleri A.S. (Turkey)"},
	{"178.250.88.190", "AS29399 - Ramtek Telekomunikasyon Hizmetleri Sanayi Ve Ticaret Limited Sirketi (Turkey, Yalova)"},
	{"88.250.63.46", "AS47331 - Turk Telekom (Turkey, Van)"},
	{"78.189.189.80", "AS9121 - Turk Telekom (Turkey, Atakum)"},
	{"88.250.204.241", "AS9121 - Turk Telekom (Turkey, Magnesia ad Sipylum)"},
	{"78.188.74.191", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"95.9.84.66", "AS9121 - Turk Telekom (Turkey, Kayseri)"},
	{"94.73.154.205", "AS34619 - Cizgi Telekomunikasyon Anonim Sirketi (Turkey)"},
	{"95.173.162.122", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"89.252.129.2", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey, Istanbul)"},
	{"95.173.184.115", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"185.169.183.130", "AS206119 - Veganet Teknolojileri ve Hizmetleri LTD STI (Turkey, Istanbul)"},
	{"79.98.134.213", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"212.64.215.122", "AS61135 - Comnet Bilgi Iletisim Teknolojileri Ticaret A.s. (Turkey)"},
	{"85.105.216.109", "AS9121 - Turk Telekom (Turkey, Konya)"},
	{"91.230.149.174", "AS212301 - Makdos Bilisim Teknolojileri Sanayi Ticaret Limited Sirketi (Turkey)"},
	{"81.215.2.60", "AS47331 - Turk Telekom (Turkey, Ankara)"},
	{"79.98.134.214", "AS42926 - Radore Veri Merkezi Hizmetleri A.S. (Turkey)"},
	{"91.93.153.73", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"81.214.70.155", "AS9121 - Turk Telekom (Turkey, Samsun)"},
	{"95.173.162.75", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"195.46.129.72", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Alanya)"},
	{"88.249.37.14", "AS47331 - Turk Telekom (Turkey, Batman)"},
	{"194.27.101.249", "AS211249 - Yildiz Teknik Universitesi (Turkey)"},
	{"95.0.124.163", "AS9121 - Turk Telekom (Turkey)"},
	{"85.105.252.11", "AS47331 - Turk Telekom (Turkey, Ankara)"},
	{"78.186.58.171", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"78.186.191.47", "AS9121 - Turk Telekom (Turkey, Bursa)"},
	{"81.214.12.7", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"88.247.12.187", "AS9121 - Turk Telekom (Turkey, Ankara)"},
	{"195.33.213.254", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Adapazarı)"},
	{"78.186.166.25", "AS9121 - Turk Telekom (Turkey, Bursa)"},
	{"84.44.32.43", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Istanbul)"},
	{"81.214.85.50", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"88.249.65.185", "AS9121 - Turk Telekom (Turkey, Yusufeli)"},
	{"78.189.87.241", "AS47331 - Turk Telekom (Turkey, Istanbul)"},
	{"77.223.128.220", "AS43391 - Netdirekt A.S. (Turkey)"},
	{"77.223.128.222", "AS43391 - Netdirekt A.S. (Turkey)"},
	{"85.105.160.195", "AS47331 - Turk Telekom (Turkey, Kartal)"},
	{"78.188.42.52", "AS47331 - Turk Telekom (Turkey, Istanbul)"},
	{"212.154.19.244", "AS12735 - TurkNet Iletisim Hizmetleri A.S. (Turkey, Istanbul)"},
	{"81.8.106.45", "AS15924 - Vodafone Net Iletisim Hizmetleri Anonim Sirketi (Turkey, Kayseri)"},
	{"212.57.11.190", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey, Mugla)"},
	{"81.214.190.74", "AS47331 - Turk Telekom (Turkey, Izmir)"},
	{"78.186.163.250", "AS47331 - Turk Telekom (Turkey, Pamukkale)"},
	{"81.214.54.88", "AS9121 - Turk Telekom (Turkey, Istanbul)"},
	{"176.236.31.156", "AS34984 - Superonline Iletisim Hizmetleri A.S. (Turkey)"},
	{"94.102.2.15", "AS51559 - Netinternet Bilisim Teknolojileri AS (Turkey)"},
	{"78.186.147.181", "AS47331 - Turk Telekom (Turkey, Istanbul)"},
	{"85.99.244.74", "AS47331 - Turk Telekom (Turkey, Izmir)"},
	{"85.105.81.252", "AS47331 - Turk Telekom (Turkey, Istanbul)"},
	{"78.186.191.171", "AS47331 - Turk Telekom (Turkey, Bursa)"},
	{"188.132.203.3", "AS202561 - High Speed Telekomunikasyon ve Hab. Hiz. Ltd. Sti. (Turkey, Antakya)"},
	{"78.135.102.232", "AS8685 - Doruk Iletisim ve Otomasyon Sanayi ve Ticaret A.S. (Turkey)"},
}*/

func main() {
	var (
		listFile    = flag.String("list", "", "DNS server list file (optional)")
		domainsFile = flag.String("domains", "", "Domain list file (optional)")
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

	// Load domains
	var domains []DomainCategory
	if *domainsFile != "" {
		domainsFromFile, err := loadDomainsFromFile(*domainsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading domains from file: %v\n", err)
			os.Exit(1)
		}
		domains = domainsFromFile
		fmt.Fprintf(os.Stderr, "Using domains from file: %s\n", *domainsFile)
	} else {
		domains = defaultDomains
		fmt.Fprintf(os.Stderr, "Using default domains list\n")
	}

	fmt.Fprintf(os.Stderr, "Testing %d DNS servers against %d domains...\n", len(dnsServers), len(domains))

	// Run tests
	results := runDNSTests(dnsServers, domains, time.Duration(*timeoutFlag)*time.Second, *workersFlag)

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
	fmt.Println("  --domains <file>   Domain list file (domain per line, optional category after space)")
	fmt.Println("  --output <file>    Output file for results (default: stdout)")
	fmt.Printf("  --format <format>  Output format: json, text (default: %s)\n", DefaultFormat)
	fmt.Printf("  --timeout <sec>    Timeout for DNS queries in seconds (default: %d)\n", DefaultTimeout)
	fmt.Printf("  --workers <num>    Number of concurrent workers (default: %d)\n", DefaultWorkerCount)
	fmt.Println("  --help            Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  go run . --list ./dns-servers.txt --domains ./domains.txt --output ./results.json")
	fmt.Println("  go run . --output ./results.txt --format text")
	fmt.Println("  go run .  (uses default DNS servers and domains)")
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

func loadDomainsFromFile(filename string) ([]DomainCategory, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var domains []DomainCategory
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

		domain := parts[0]
		category := CategoryOther // Default category
		
		if len(parts) > 1 {
			switch strings.ToLower(parts[1]) {
			case "general":
				category = CategoryGeneral
			case "ad-server", "adserver":
				category = CategoryAdServer
			case "adult":
				category = CategoryAdult
			case "other":
				category = CategoryOther
			default:
				category = CategoryOther
			}
		}

		domains = append(domains, DomainCategory{Domain: domain, Category: category})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return domains, nil
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
