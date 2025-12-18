package ti

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"gateway-server/internal/models"
)

type IOC struct {
	ID       string `json:"id"`
	Type     string `json:"ioc_type"`
	Value    string `json:"value"`
	FeedID   string `json:"feed_id"`
	FeedName string `json:"feed_name"`
}

type Feed struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type TIClient struct {
	serverURL  string
	cache      *IOCCache
	httpClient *http.Client
}

type IOCCache struct {
	mu         sync.RWMutex
	iocsByFeed map[string][]IOC
	feedNames  map[string]string
	lastUpdate time.Time
}

func NewTIClient(serverURL string) *TIClient {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	return &TIClient{
		serverURL:  serverURL,
		httpClient: httpClient,
		cache: &IOCCache{
			iocsByFeed: make(map[string][]IOC),
			feedNames:  make(map[string]string),
		},
	}
}

func (c *TIClient) GetCache() *IOCCache {
	return c.cache
}

func (c *TIClient) GetServerURL() string {
	return c.serverURL
}

func (c *TIClient) FetchFeed(feedID string) (*Feed, error) {
	return c.FetchFeedWithKey(feedID, "")
}

func (c *TIClient) FetchFeedWithKey(feedID string, apiKey string) (*Feed, error) {
	url := fmt.Sprintf("%s/feeds/%s", c.serverURL, feedID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var feed Feed
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal feed: %w", err)
	}

	return &feed, nil
}

func (c *TIClient) FetchIOCsForFeed(feedID string) ([]IOC, error) {
	return c.FetchIOCsForFeedWithKey(feedID, "")
}

func (c *TIClient) FetchIOCsForFeedWithKey(feedID string, apiKey string) ([]IOC, error) {
	url := fmt.Sprintf("%s/feeds/%s/iocs", c.serverURL, feedID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch IOCs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var iocs []IOC
	if err := json.Unmarshal(body, &iocs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal IOCs: %w", err)
	}

	return iocs, nil
}

type FeedWithKey struct {
	FeedID string
	APIKey string
}

func (c *TIClient) UpdateCache(feedIDs []string) error {
	feeds := make([]FeedWithKey, len(feedIDs))
	for i, id := range feedIDs {
		feeds[i] = FeedWithKey{FeedID: id, APIKey: ""}
	}
	return c.UpdateCacheWithKeys(feeds)
}

func (c *TIClient) UpdateCacheWithKeys(feeds []FeedWithKey) error {
	c.cache.mu.Lock()
	defer c.cache.mu.Unlock()

	if c.cache.iocsByFeed == nil {
		c.cache.iocsByFeed = make(map[string][]IOC)
	}
	if c.cache.feedNames == nil {
		c.cache.feedNames = make(map[string]string)
	}

	for _, feedInfo := range feeds {
		feed, err := c.FetchFeedWithKey(feedInfo.FeedID, feedInfo.APIKey)
		if err != nil {
			continue
		}
		c.cache.feedNames[feedInfo.FeedID] = feed.Name

		iocs, err := c.FetchIOCsForFeedWithKey(feedInfo.FeedID, feedInfo.APIKey)
		if err != nil {
			continue
		}

		for i := range iocs {
			iocs[i].FeedName = feed.Name
		}

		c.cache.iocsByFeed[feedInfo.FeedID] = iocs
	}

	c.cache.lastUpdate = time.Now()

	return nil
}

func (c *IOCCache) GetIOCsForFeeds(feedIDs []string) []IOC {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var allIOCs []IOC
	for _, feedID := range feedIDs {
		if iocs, ok := c.iocsByFeed[feedID]; ok {
			allIOCs = append(allIOCs, iocs...)
		}
	}

	return allIOCs
}

func AnalyzeRequest(req *http.Request, iocs []IOC) []models.IOCMatch {
	var matches []models.IOCMatch

	var requestText strings.Builder

	requestText.WriteString(req.URL.Path)
	requestText.WriteString(" ")
	requestText.WriteString(req.URL.RawQuery)
	requestText.WriteString(" ")

	for key, values := range req.Header {
		requestText.WriteString(key)
		requestText.WriteString(": ")
		requestText.WriteString(strings.Join(values, " "))
		requestText.WriteString(" ")
	}

	clientIP := getClientIP(req)
	requestText.WriteString(clientIP)
	requestText.WriteString(" ")

	if req.Body != nil {
		bodyBytes, err := io.ReadAll(io.LimitReader(req.Body, 10*1024))
		if err == nil {
			requestText.Write(bodyBytes)
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	fullText := requestText.String()

	for _, ioc := range iocs {
		if strings.Contains(fullText, ioc.Value) {
			matches = append(matches, models.IOCMatch{
				IOCType:  ioc.Type,
				IOCValue: ioc.Value,
				Location: "request",
				FeedID:   ioc.FeedID,
				FeedName: ioc.FeedName,
			})
		}
	}

	return matches
}

func getClientIP(req *http.Request) string {
	xff := req.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	xri := req.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	ip, _, _ := net.SplitHostPort(req.RemoteAddr)
	return ip
}

func matchIP(clientIP, iocIP string) bool {
	if clientIP == iocIP {
		return true
	}

	_, ipNet, err := net.ParseCIDR(iocIP)
	if err != nil {
		return false
	}

	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	return ipNet.Contains(ip)
}
