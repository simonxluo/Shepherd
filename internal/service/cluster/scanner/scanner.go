// Package scanner provides network scanning capabilities for client discovery.
package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/utils"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/config"
	"github.com/shepherd-project/shepherd/Shepherd/internal/comm/logger"
	"github.com/shepherd-project/shepherd/Shepherd/internal/service/cluster"
)

// Scanner performs network scans to discover client nodes
type Scanner struct {
	config     *config.NetworkScanConfig
	clients    []*cluster.DiscoveredClient
	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	scanning   bool
	log        *logger.Logger
	httpClient HTTPClient
}

// HTTPClient interface for making HTTP requests (for testability)
type HTTPClient interface {
	Get(url string) (int, []byte, error)
}

// DefaultHTTPClient implements HTTPClient using standard HTTP
type DefaultHTTPClient struct {
	client *http.Client
}

// NewDefaultHTTPClient creates a new HTTP client
func NewDefaultHTTPClient(timeout time.Duration) *DefaultHTTPClient {
	return &DefaultHTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Get performs a GET request
func (c *DefaultHTTPClient) Get(url string) (int, []byte, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return 0, nil, err
	}
	defer utils.CloseQuietly(resp.Body)

	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	return resp.StatusCode, body[:n], nil
}

// NewScanner creates a new network scanner
func NewScanner(cfg *config.NetworkScanConfig, log *logger.Logger) *Scanner {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scanner{
		config:     cfg,
		clients:    make([]*cluster.DiscoveredClient, 0),
		ctx:        ctx,
		cancel:     cancel,
		scanning:   false,
		log:        log,
		httpClient: NewDefaultHTTPClient(time.Duration(cfg.Timeout) * time.Second),
	}
}

// Start starts the scanner
func (s *Scanner) Start() {
	if s.config.Interval > 0 {
		s.wg.Add(1)
		go s.autoScan()
	}
}

// Stop stops the scanner
func (s *Scanner) Stop() {
	s.cancel()
	s.wg.Wait()
}

// Scan performs a network scan
func (s *Scanner) Scan() ([]*cluster.DiscoveredClient, error) {
	s.mu.Lock()
	if s.scanning {
		s.mu.Unlock()
		return nil, fmt.Errorf("扫描正在进行中")
	}
	s.scanning = true
	s.clients = make([]*cluster.DiscoveredClient, 0)
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.scanning = false
		s.mu.Unlock()
	}()

	s.log.Info("开始网络扫描", nil)

	var wg sync.WaitGroup
	results := make(chan *cluster.DiscoveredClient, 100)
	errors := make(chan error, 100)

	for _, subnet := range s.config.Subnets {
		wg.Add(1)
		go func(subnet string) {
			defer wg.Done()
			s.scanSubnet(subnet, results, errors)
		}(subnet)
	}

	// Wait for all scans to complete
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	// Collect results
	discovered := make([]*cluster.DiscoveredClient, 0)
	for client := range results {
		discovered = append(discovered, client)
	}

	// Collect errors
	errorCount := 0
	for range errors {
		errorCount++
	}

	s.mu.Lock()
	s.clients = discovered
	s.mu.Unlock()

	s.log.Info(fmt.Sprintf("网络扫描完成: 发现 %d 个客户端 (%d 个错误)", len(discovered), errorCount), nil)

	return discovered, nil
}

// GetStatus returns the current scan status
func (s *Scanner) GetStatus() *cluster.ScanStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := make([]cluster.DiscoveredClient, len(s.clients))
	for i, c := range s.clients {
		found[i] = *c
	}

	return &cluster.ScanStatus{
		Running: s.scanning,
		Found:   found,
	}
}

// scanSubnet scans a single subnet for clients
func (s *Scanner) scanSubnet(subnet string, results chan<- *cluster.DiscoveredClient, errors chan<- error) {
	// Parse subnet (e.g., "192.168.1.0/24")
	ip, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		s.log.Error(fmt.Sprintf("解析子网失败: %s - %v", subnet, err), nil)
		errors <- err
		return
	}

	// Parse port range
	startPort, endPort, err := s.parsePortRange(s.config.PortRange)
	if err != nil {
		s.log.Error(fmt.Sprintf("解析端口范围失败: %s - %v", s.config.PortRange, err), nil)
		errors <- err
		return
	}

	// Iterate through all IPs in subnet
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		// Skip network and broadcast addresses
		if ip.IsUnspecified() || ip.IsLoopback() {
			continue
		}

		// Scan each port in range
		for port := startPort; port <= endPort; port++ {
			select {
			case <-s.ctx.Done():
				return
			default:
				// Check each host/port combination
				go func(host string, portNum int) {
					if client := s.checkClient(host, portNum); client != nil {
						results <- client
					}
				}(ip.String(), port)
			}
		}
	}
}

// checkClient checks if a client is running at the given address
func (s *Scanner) checkClient(host string, port int) *cluster.DiscoveredClient {
	url := fmt.Sprintf("http://%s:%d/api/client/status", host, port)

	statusCode, body, err := s.httpClient.Get(url)
	if err != nil {
		return nil // Not a client, or not reachable
	}

	if statusCode != 200 {
		return nil
	}

	// Parse response
	var info struct {
		ID           string                `json:"id"`
		Name         string                `json:"name"`
		Version      string                `json:"version"`
		Capabilities *cluster.Capabilities `json:"capabilities"`
		Tags         []string              `json:"tags"`
	}

	if len(body) > 0 {
		if err := json.Unmarshal(body, &info); err != nil {
			s.log.Debug(fmt.Sprintf("解析客户端信息失败: %s:%d - %v", host, port, err))
			return nil
		}
	}

	return &cluster.DiscoveredClient{
		Address:      host,
		Port:         port,
		ID:           info.ID,
		Name:         info.Name,
		Version:      info.Version,
		Capabilities: info.Capabilities,
		Tags:         info.Tags,
	}
}

// parsePortRange parses a port range string (e.g., "9191-9200")
func (s *Scanner) parsePortRange(portRange string) (int, int, error) {
	parts := strings.Split(portRange, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("无效的端口范围格式: %s", portRange)
	}

	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("无效的起始端口: %s", parts[0])
	}

	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("无效的结束端口: %s", parts[1])
	}

	if start < 1 || start > 65535 || end < 1 || end > 65535 || start > end {
		return 0, 0, fmt.Errorf("端口范围无效: %d-%d", start, end)
	}

	return start, end, nil
}

// inc increments an IP address
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// autoScan performs automatic scans at configured intervals
func (s *Scanner) autoScan() {
	defer s.wg.Done()

	if s.config.Interval <= 0 {
		return
	}

	ticker := time.NewTicker(time.Duration(s.config.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if s.config.AutoDiscover {
				func() {
					_, err := s.Scan()
					if err != nil {
						s.log.Warn("模型扫描失败", "error", err)
					}
				}()
			}
		}
	}
}
