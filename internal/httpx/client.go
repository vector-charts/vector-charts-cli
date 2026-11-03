package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zydro-marine/vector-charts-cli/internal/config"
)

type Client struct {
	HTTP      *http.Client
	PortalURL string
	APIURL    string
	Type      string
	Session   string
	APIToken  string
}

func New(profile config.Profile, creds config.Creds) *Client {
	p := profile.Normalized()
	return &Client{
		HTTP: &http.Client{
			Timeout: 120 * time.Second,
		},
		PortalURL: p.URL,
		APIURL:    p.APIURL,
		Type:      p.Type,
		Session:   creds.SessionToken,
		APIToken:  creds.APIToken,
	}
}

func (c *Client) portalBearer() string {
	// Personal access tokens (preferred) work on cloud and OEM portal.
	if c.APIToken != "" {
		return c.APIToken
	}
	return c.Session
}

type APIError struct {
	Status  int
	Message string
	Body    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("HTTP %d: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("HTTP %d: %s", e.Status, truncate(e.Body, 200))
}

// IsConnectionError reports transport failures (dial/timeout/DNS), not HTTP responses.
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	return !errors.As(err, &apiErr)
}

// IsAuthError reports HTTP 401/403 responses from the portal.
func IsAuthError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Status == http.StatusUnauthorized || apiErr.Status == http.StatusForbidden
}

// ClassifyWhoAmIError maps whoami failures to user-facing auth/connect errors.
func ClassifyWhoAmIError(err error) error {
	if err == nil {
		return nil
	}
	if IsConnectionError(err) {
		return fmt.Errorf("failed to connect")
	}
	if IsAuthError(err) {
		return fmt.Errorf("authentication expired, please login again")
	}
	return fmt.Errorf("failed to connect")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (c *Client) doJSON(method, rawURL, bearer string, in any, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := ""
		var parsed map[string]any
		if json.Unmarshal(data, &parsed) == nil {
			if m, ok := parsed["message"].(string); ok {
				msg = m
			}
		}
		return &APIError{Status: resp.StatusCode, Message: msg, Body: string(data)}
	}
	if out == nil || resp.StatusCode == 204 || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

func (c *Client) PortalJSON(method, path string, in, out any) error {
	u := c.PortalURL + "/api/portal" + path
	return c.doJSON(method, u, c.portalBearer(), in, out)
}

// DefaultListPageSize matches the portal API max page size for cells/jobs.
const DefaultListPageSize = 200

func withLimitOffset(path string, limit, offset int) string {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%slimit=%d&offset=%d", path, sep, limit, offset)
}

func paginationTotal(page map[string]any) int {
	pag, _ := page["pagination"].(map[string]any)
	if pag == nil {
		return 0
	}
	switch v := pag["total"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}

func extractPageItems(page map[string]any, listKey string) []any {
	if page == nil {
		return nil
	}
	if arr, ok := page[listKey].([]any); ok {
		return arr
	}
	return nil
}

// PortalListAll fetches every page of a limit/offset list endpoint and returns
// the combined items under listKey. Tables should print the full set in one go.
func (c *Client) PortalListAll(path, listKey string, pageSize int) (map[string]any, []any, error) {
	if pageSize <= 0 {
		pageSize = DefaultListPageSize
	}
	var all []any
	offset := 0
	for {
		var page map[string]any
		if err := c.PortalJSON("GET", withLimitOffset(path, pageSize, offset), nil, &page); err != nil {
			return nil, nil, err
		}
		items := extractPageItems(page, listKey)
		all = append(all, items...)
		total := paginationTotal(page)
		if len(items) == 0 {
			break
		}
		if total > 0 && len(all) >= total {
			break
		}
		if len(items) < pageSize {
			break
		}
		offset += pageSize
	}
	out := map[string]any{
		listKey: all,
		"pagination": map[string]any{
			"limit":  len(all),
			"offset": 0,
			"total":  len(all),
		},
	}
	return out, all, nil
}

func (c *Client) PortalJSONPublic(method, path string, in, out any) error {
	u := c.PortalURL + "/api/portal" + path
	return c.doJSON(method, u, "", in, out)
}

func (c *Client) ServerJSON(method, path string, in, out any) error {
	u := c.APIURL + path
	return c.doJSON(method, u, c.APIToken, in, out)
}

func (c *Client) ServerJSONPublic(method, path string, in, out any) error {
	u := c.APIURL + path
	return c.doJSON(method, u, "", in, out)
}

func (c *Client) PortalHealth() (map[string]any, error) {
	out := map[string]any{}
	err := c.PortalJSONPublic("GET", "/v1/healthcheck", nil, &out)
	return out, err
}

func (c *Client) ServerHealth() (map[string]any, error) {
	out := map[string]any{}
	err := c.ServerJSONPublic("GET", "/api/v1/healthcheck", nil, &out)
	return out, err
}

func (c *Client) WhoAmI() (map[string]any, error) {
	out := map[string]any{}
	err := c.PortalJSON("GET", "/v1/auth/whoami", nil, &out)
	return out, err
}

func (c *Client) Login(username, password string) (tokenID string, user map[string]any, err error) {
	var out struct {
		Token struct {
			ID string `json:"id"`
		} `json:"token"`
		User map[string]any `json:"user"`
	}
	err = c.PortalJSONPublic("POST", "/v1/auth/login", map[string]string{
		"username": username,
		"password": password,
	}, &out)
	if err != nil {
		return "", nil, err
	}
	return out.Token.ID, out.User, nil
}

func (c *Client) IssuePersonalAccessToken(machineID, machineLabel string) (tokenID string, user map[string]any, err error) {
	body := map[string]any{"machineId": machineID}
	if machineLabel != "" {
		body["machineLabel"] = machineLabel
	}
	var out struct {
		Token struct {
			ID string `json:"id"`
		} `json:"token"`
		User map[string]any `json:"user"`
	}
	err = c.PortalJSON("POST", "/v1/auth/personal-access-token", body, &out)
	if err != nil {
		return "", nil, err
	}
	return out.Token.ID, out.User, nil
}

func (c *Client) Logout() error {
	return c.PortalJSON("POST", "/v1/auth/logout", nil, nil)
}

type DeviceCodeStart struct {
	DeviceCode              string `json:"deviceCode"`
	UserCode                string `json:"userCode"`
	VerificationURI         string `json:"verificationUri"`
	VerificationURIComplete string `json:"verificationUriComplete"`
	ExpiresIn               int    `json:"expiresIn"`
	Interval                int    `json:"interval"`
}

func (c *Client) StartDeviceAuth(machineID, machineLabel string) (*DeviceCodeStart, error) {
	body := map[string]any{
		"machineId":     machineID,
		"portalBaseUrl": c.PortalURL,
	}
	if machineLabel != "" {
		body["machineLabel"] = machineLabel
	}
	var out DeviceCodeStart
	err := c.PortalJSONPublic("POST", "/v1/auth/device/code", body, &out)
	if err != nil {
		return nil, err
	}
	// Always open the device page on this profile's portal URL. Older OEM
	// portals ignore portalBaseUrl and return email.baseUri (often :5173).
	if out.UserCode != "" {
		q := url.Values{}
		q.Set("userCode", out.UserCode)
		q.Set("machineId", machineID)
		if machineLabel != "" {
			q.Set("machineLabel", machineLabel)
		}
		out.VerificationURI = strings.TrimRight(c.PortalURL, "/") + "/device"
		out.VerificationURIComplete = out.VerificationURI + "?" + q.Encode()
	}
	if out.Interval <= 0 {
		out.Interval = 5
	}
	return &out, nil
}

func (c *Client) PollDeviceAuth(deviceCode string) (status string, tokenID string, user map[string]any, err error) {
	var out struct {
		Status string `json:"status"`
		Token  struct {
			ID string `json:"id"`
		} `json:"token"`
		User map[string]any `json:"user"`
	}
	err = c.PortalJSONPublic("POST", "/v1/auth/device/token", map[string]string{
		"deviceCode": deviceCode,
	}, &out)
	if err != nil {
		return "", "", nil, err
	}
	return out.Status, out.Token.ID, out.User, nil
}

func (c *Client) UploadChart(filePath string) (map[string]any, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.PortalURL+"/api/portal/v1/charts/upload", &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if bearer := c.portalBearer(); bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{Status: resp.StatusCode, Body: string(data)}
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if upload, ok := out["upload"].(map[string]any); ok {
		return upload, nil
	}
	return out, nil
}
