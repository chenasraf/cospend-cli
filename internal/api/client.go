package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/chenasraf/cospend-cli/internal/config"
)

// Client is the Cospend API client
type Client struct {
	config      *config.Config
	httpClient  *http.Client
	Debug       bool
	DebugWriter io.Writer
}

// Member represents a project member
type Member struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	UserID    string `json:"userid"`
	Activated bool   `json:"activated"`
}

// Category represents a bill category
type Category struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

// PaymentMode represents a payment method
type PaymentMode struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

// Currency represents a currency
type Currency struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	ExchangeRate float64 `json:"exchange_rate"`
}

// Project represents a Cospend project
type Project struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	CurrencyName string        `json:"currencyname"`
	Members      []Member      `json:"members"`
	Categories   []Category    // custom unmarshal
	PaymentModes []PaymentMode // custom unmarshal
	Currencies   []Currency    `json:"currencies"`
}

// UnmarshalJSON custom unmarshaler to handle categories/paymentmodes as object or array
func (p *Project) UnmarshalJSON(data []byte) error {
	// Use a map for flexible parsing
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Parse simple fields
	if v, ok := raw["id"]; ok {
		_ = json.Unmarshal(v, &p.ID)
	}
	if v, ok := raw["name"]; ok {
		_ = json.Unmarshal(v, &p.Name)
	}
	if v, ok := raw["currencyname"]; ok {
		_ = json.Unmarshal(v, &p.CurrencyName)
	}
	if v, ok := raw["members"]; ok {
		_ = json.Unmarshal(v, &p.Members)
	}
	if v, ok := raw["currencies"]; ok {
		_ = json.Unmarshal(v, &p.Currencies)
	}

	// Parse categories (can be array or object)
	if v, ok := raw["categories"]; ok {
		p.Categories = parseCategories(v)
	}

	// Parse payment modes (can be array or object)
	if v, ok := raw["paymentmodes"]; ok {
		p.PaymentModes = parsePaymentModes(v)
	}

	return nil
}

// MarshalJSON custom marshaler for Project
func (p Project) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID           string        `json:"id"`
		Name         string        `json:"name"`
		CurrencyName string        `json:"currencyname"`
		Members      []Member      `json:"members"`
		Categories   []Category    `json:"categories"`
		PaymentModes []PaymentMode `json:"paymentmodes"`
		Currencies   []Currency    `json:"currencies"`
	}{
		ID:           p.ID,
		Name:         p.Name,
		CurrencyName: p.CurrencyName,
		Members:      p.Members,
		Categories:   p.Categories,
		PaymentModes: p.PaymentModes,
		Currencies:   p.Currencies,
	})
}

func parseCategories(data json.RawMessage) []Category {
	// API returns categories as object keyed by ID
	var obj map[string]Category
	if err := json.Unmarshal(data, &obj); err == nil {
		result := make([]Category, 0, len(obj))
		for idStr, cat := range obj {
			if id, err := strconv.Atoi(idStr); err == nil {
				cat.ID = id
			}
			result = append(result, cat)
		}
		return result
	}
	// Fallback to array (for tests)
	var arr []Category
	if err := json.Unmarshal(data, &arr); err == nil {
		return arr
	}
	return nil
}

func parsePaymentModes(data json.RawMessage) []PaymentMode {
	// API returns payment modes as object keyed by ID
	var obj map[string]PaymentMode
	if err := json.Unmarshal(data, &obj); err == nil {
		result := make([]PaymentMode, 0, len(obj))
		for idStr, pm := range obj {
			if id, err := strconv.Atoi(idStr); err == nil {
				pm.ID = id
			}
			result = append(result, pm)
		}
		return result
	}
	// Fallback to array (for tests)
	var arr []PaymentMode
	if err := json.Unmarshal(data, &arr); err == nil {
		return arr
	}
	return nil
}

// Bill represents a bill to create
type Bill struct {
	What               string  `json:"what"`
	Amount             float64 `json:"amount"`
	PayerID            int     `json:"payer_id"`
	OwedTo             []int   `json:"-"` // Will be formatted as comma-separated string
	Date               string  `json:"date"`
	Comment            string  `json:"comment,omitempty"`
	PaymentModeID      int     `json:"paymentmodeid,omitempty"`
	CategoryID         int     `json:"categoryid,omitempty"`
	OriginalCurrencyID int     `json:"original_currency_id,omitempty"`
}

// BillResponse represents a bill returned from the API
type BillResponse struct {
	ID            int     `json:"id"`
	What          string  `json:"what"`
	Amount        float64 `json:"amount"`
	Date          string  `json:"date"`
	PayerID       int     `json:"payer_id"`
	Owers         []Ower  `json:"owers"`
	Comment       string  `json:"comment"`
	PaymentModeID int     `json:"paymentmodeid"`
	CategoryID    int     `json:"categoryid"`
	Repeat        string  `json:"repeat"`
	Timestamp     int64   `json:"timestamp"`
}

// Ower represents a member who owes part of a bill
type Ower struct {
	ID     int     `json:"id"`
	Weight float64 `json:"weight"`
}

// OCSResponse wraps the OCS API response format
type OCSResponse struct {
	OCS struct {
		Meta struct {
			Status     string `json:"status"`
			StatusCode int    `json:"statuscode"`
			Message    string `json:"message"`
		} `json:"meta"`
		Data json.RawMessage `json:"data"`
	} `json:"ocs"`
}

// NewClient creates a new API client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		config:     cfg,
		httpClient: &http.Client{},
	}
}

func (c *Client) debugf(format string, args ...interface{}) {
	if c.Debug && c.DebugWriter != nil {
		_, _ = fmt.Fprintf(c.DebugWriter, "[DEBUG] "+format+"\n", args...)
	}
}

func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	baseURL := config.NormalizeURL(c.config.Domain)
	fullURL := fmt.Sprintf("%s%s", baseURL, path)

	c.debugf("Request: %s %s", method, fullURL)

	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.SetBasicAuth(c.config.User, c.config.Password)
	req.Header.Set("OCS-APIRequest", "true")
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	c.debugf("Headers: OCS-APIRequest=true, Accept=application/json, Auth=Basic %s:***", c.config.User)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.debugf("Request error: %v", err)
		return nil, err
	}

	c.debugf("Response: %d %s", resp.StatusCode, resp.Status)

	return resp, nil
}

// GetProject fetches project details including members, categories, and payment modes
func (c *Client) GetProject(projectID string) (*Project, error) {
	path := fmt.Sprintf("/ocs/v2.php/apps/cospend/api/v1/projects/%s", url.PathEscape(projectID))

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching project: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ocsResp OCSResponse
	if err := json.NewDecoder(resp.Body).Decode(&ocsResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if ocsResp.OCS.Meta.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %s", ocsResp.OCS.Meta.Message)
	}

	var project Project
	if err := json.Unmarshal(ocsResp.OCS.Data, &project); err != nil {
		return nil, fmt.Errorf("decoding project data: %w", err)
	}

	return &project, nil
}

// ProjectSummary represents a project in the list response
type ProjectSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CurrName   string `json:"currencyname"`
	ArchivedTS *int64 `json:"archived_ts"`
}

// IsArchived returns true if the project is archived
func (p *ProjectSummary) IsArchived() bool {
	return p.ArchivedTS != nil
}

// GetProjects fetches all projects the user has access to
func (c *Client) GetProjects() ([]ProjectSummary, error) {
	path := "/ocs/v2.php/apps/cospend/api/v1/projects"

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching projects: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ocsResp OCSResponse
	if err := json.NewDecoder(resp.Body).Decode(&ocsResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if ocsResp.OCS.Meta.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %s", ocsResp.OCS.Meta.Message)
	}

	c.debugf("Projects response: %s", string(ocsResp.OCS.Data))

	var projects []ProjectSummary
	if err := json.Unmarshal(ocsResp.OCS.Data, &projects); err != nil {
		return nil, fmt.Errorf("decoding projects data: %w", err)
	}

	return projects, nil
}

// CreateBill creates a new bill in the project
func (c *Client) CreateBill(projectID string, bill Bill) error {
	path := fmt.Sprintf("/ocs/v2.php/apps/cospend/api/v1/projects/%s/bills", url.PathEscape(projectID))

	// Build form data
	data := url.Values{}
	data.Set("what", bill.What)
	data.Set("amount", strconv.FormatFloat(bill.Amount, 'f', 2, 64))
	data.Set("payer", strconv.Itoa(bill.PayerID))
	data.Set("date", bill.Date)
	data.Set("timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	data.Set("repeat", "n")

	// Format owed member IDs as comma-separated string
	owedIDs := make([]string, len(bill.OwedTo))
	for i, id := range bill.OwedTo {
		owedIDs[i] = strconv.Itoa(id)
	}
	data.Set("payedFor", strings.Join(owedIDs, ","))

	if bill.Comment != "" {
		data.Set("comment", bill.Comment)
	}
	if bill.PaymentModeID != 0 {
		data.Set("paymentmodeid", strconv.Itoa(bill.PaymentModeID))
	}
	if bill.CategoryID != 0 {
		data.Set("categoryid", strconv.Itoa(bill.CategoryID))
	}
	if bill.OriginalCurrencyID != 0 {
		data.Set("original_currency_id", strconv.Itoa(bill.OriginalCurrencyID))
	}

	c.debugf("Request body: %s", data.Encode())

	resp, err := c.doRequest("POST", path, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("creating bill: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ocsResp OCSResponse
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&ocsResp); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if ocsResp.OCS.Meta.StatusCode != 200 {
		return fmt.Errorf("API error: %s", ocsResp.OCS.Meta.Message)
	}

	return nil
}

// GetBills fetches all bills for a project
func (c *Client) GetBills(projectID string) ([]BillResponse, error) {
	path := fmt.Sprintf("/ocs/v2.php/apps/cospend/api/v1/projects/%s/bills", url.PathEscape(projectID))

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching bills: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ocsResp OCSResponse
	if err := json.NewDecoder(resp.Body).Decode(&ocsResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if ocsResp.OCS.Meta.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %s", ocsResp.OCS.Meta.Message)
	}

	// API returns: {"nb_bills": N, "bills": [...], "allBillIds": [...], "timestamp": N}
	var billsWrapper struct {
		Bills []BillResponse `json:"bills"`
	}
	if err := json.Unmarshal(ocsResp.OCS.Data, &billsWrapper); err != nil {
		return nil, fmt.Errorf("decoding bills data: %w", err)
	}

	return billsWrapper.Bills, nil
}

// UserInfo represents Nextcloud user information
type UserInfo struct {
	Locale   string `json:"locale"`
	Language string `json:"language"`
}

// GetUserInfo fetches the authenticated user's info from Nextcloud OCS API
func (c *Client) GetUserInfo() (*UserInfo, error) {
	resp, err := c.doRequest("GET", "/ocs/v2.php/cloud/user", nil)
	if err != nil {
		return nil, fmt.Errorf("fetching user info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ocsResp OCSResponse
	if err := json.NewDecoder(resp.Body).Decode(&ocsResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if ocsResp.OCS.Meta.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %s", ocsResp.OCS.Meta.Message)
	}

	var userInfo UserInfo
	if err := json.Unmarshal(ocsResp.OCS.Data, &userInfo); err != nil {
		return nil, fmt.Errorf("decoding user info: %w", err)
	}

	return &userInfo, nil
}

// DeleteBill deletes a bill from the project
func (c *Client) DeleteBill(projectID string, billID int) error {
	path := fmt.Sprintf("/ocs/v2.php/apps/cospend/api/v1/projects/%s/bills/%d", url.PathEscape(projectID), billID)

	resp, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return fmt.Errorf("deleting bill: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var ocsResp OCSResponse
	if err := json.NewDecoder(resp.Body).Decode(&ocsResp); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if ocsResp.OCS.Meta.StatusCode != 200 {
		return fmt.Errorf("API error: %s", ocsResp.OCS.Meta.Message)
	}

	return nil
}
