package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chenasraf/cospend-cli/internal/config"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		Domain:   "https://cloud.example.com",
		User:     "testuser",
		Password: "testpass",
	}

	client := NewClient(cfg)

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client.config != cfg {
		t.Error("NewClient() config not set correctly")
	}
	if client.httpClient == nil {
		t.Error("NewClient() httpClient is nil")
	}
}

func TestGetProject(t *testing.T) {
	projectData := Project{
		ID:   "test-project",
		Name: "Test Project",
		Members: []Member{
			{ID: 1, Name: "Alice", UserID: "alice", Activated: true},
			{ID: 2, Name: "Bob", UserID: "bob", Activated: true},
		},
		Categories: []Category{
			{ID: 1, Name: "Food"},
			{ID: 2, Name: "Transport"},
		},
		PaymentModes: []PaymentMode{
			{ID: 1, Name: "Cash"},
			{ID: 2, Name: "Credit Card"},
		},
		Currencies: []Currency{
			{ID: 1, Name: "$", ExchangeRate: 1.0},
		},
	}

	tests := []struct {
		name           string
		projectID      string
		responseStatus int
		responseBody   any
		wantErr        bool
	}{
		{
			name:           "successful request",
			projectID:      "test-project",
			responseStatus: http.StatusOK,
			responseBody: OCSResponse{
				OCS: struct {
					Meta struct {
						Status     string `json:"status"`
						StatusCode int    `json:"statuscode"`
						Message    string `json:"message"`
					} `json:"meta"`
					Data json.RawMessage `json:"data"`
				}{
					Meta: struct {
						Status     string `json:"status"`
						StatusCode int    `json:"statuscode"`
						Message    string `json:"message"`
					}{
						Status:     "ok",
						StatusCode: 200,
						Message:    "OK",
					},
					Data: mustMarshal(projectData),
				},
			},
			wantErr: false,
		},
		{
			name:           "project not found",
			projectID:      "nonexistent",
			responseStatus: http.StatusNotFound,
			responseBody:   "Not Found",
			wantErr:        true,
		},
		{
			name:           "api error",
			projectID:      "test-project",
			responseStatus: http.StatusOK,
			responseBody: OCSResponse{
				OCS: struct {
					Meta struct {
						Status     string `json:"status"`
						StatusCode int    `json:"statuscode"`
						Message    string `json:"message"`
					} `json:"meta"`
					Data json.RawMessage `json:"data"`
				}{
					Meta: struct {
						Status     string `json:"status"`
						StatusCode int    `json:"statuscode"`
						Message    string `json:"message"`
					}{
						Status:     "failure",
						StatusCode: 404,
						Message:    "Project not found",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request headers
				if r.Header.Get("OCS-APIRequest") != "true" {
					t.Error("Missing OCS-APIRequest header")
				}

				// Verify Basic Auth
				user, pass, ok := r.BasicAuth()
				if !ok {
					t.Error("Missing Basic Auth")
				}
				if user != "testuser" || pass != "testpass" {
					t.Errorf("Wrong credentials: %s:%s", user, pass)
				}

				// Verify path
				expectedPath := "/ocs/v2.php/apps/cospend/api/v1/projects/" + tt.projectID
				if r.URL.Path != expectedPath {
					t.Errorf("Wrong path: got %s, want %s", r.URL.Path, expectedPath)
				}

				w.WriteHeader(tt.responseStatus)
				if s, ok := tt.responseBody.(string); ok {
					_, _ = w.Write([]byte(s))
				} else {
					_ = json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			cfg := &config.Config{
				Domain:   server.URL,
				User:     "testuser",
				Password: "testpass",
			}
			client := NewClient(cfg)

			project, err := client.GetProject(tt.projectID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && project != nil {
				if project.ID != projectData.ID {
					t.Errorf("GetProject() ID = %v, want %v", project.ID, projectData.ID)
				}
				if len(project.Members) != len(projectData.Members) {
					t.Errorf("GetProject() Members count = %v, want %v", len(project.Members), len(projectData.Members))
				}
			}
		})
	}
}

func TestCreateBill(t *testing.T) {
	tests := []struct {
		name           string
		bill           Bill
		responseStatus int
		responseBody   any
		wantErr        bool
		checkRequest   func(t *testing.T, r *http.Request)
	}{
		{
			name: "successful creation",
			bill: Bill{
				What:          "Test expense",
				Amount:        25.50,
				PayerID:       1,
				OwedTo:        []int{1, 2},
				Date:          "2024-01-15",
				Comment:       "Test comment",
				PaymentModeID: 1,
				CategoryID:    2,
			},
			responseStatus: http.StatusOK,
			responseBody: OCSResponse{
				OCS: struct {
					Meta struct {
						Status     string `json:"status"`
						StatusCode int    `json:"statuscode"`
						Message    string `json:"message"`
					} `json:"meta"`
					Data json.RawMessage `json:"data"`
				}{
					Meta: struct {
						Status     string `json:"status"`
						StatusCode int    `json:"statuscode"`
						Message    string `json:"message"`
					}{
						Status:     "ok",
						StatusCode: 200,
						Message:    "OK",
					},
					Data: mustMarshal(map[string]int{"id": 123}),
				},
			},
			wantErr: false,
			checkRequest: func(t *testing.T, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Wrong method: got %s, want POST", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
					t.Errorf("Wrong Content-Type: %s", r.Header.Get("Content-Type"))
				}

				_ = r.ParseForm()
				if r.FormValue("what") != "Test expense" {
					t.Errorf("Wrong what: %s", r.FormValue("what"))
				}
				if r.FormValue("amount") != "25.50" {
					t.Errorf("Wrong amount: %s", r.FormValue("amount"))
				}
				if r.FormValue("payer") != "1" {
					t.Errorf("Wrong payer: %s", r.FormValue("payer"))
				}
				if r.FormValue("payedFor") != "1,2" {
					t.Errorf("Wrong payedFor: %s", r.FormValue("payedFor"))
				}
				if r.FormValue("comment") != "Test comment" {
					t.Errorf("Wrong comment: %s", r.FormValue("comment"))
				}
			},
		},
		{
			name: "minimal bill",
			bill: Bill{
				What:    "Simple expense",
				Amount:  10.00,
				PayerID: 1,
				OwedTo:  []int{1},
				Date:    "2024-01-15",
			},
			responseStatus: http.StatusOK,
			responseBody: OCSResponse{
				OCS: struct {
					Meta struct {
						Status     string `json:"status"`
						StatusCode int    `json:"statuscode"`
						Message    string `json:"message"`
					} `json:"meta"`
					Data json.RawMessage `json:"data"`
				}{
					Meta: struct {
						Status     string `json:"status"`
						StatusCode int    `json:"statuscode"`
						Message    string `json:"message"`
					}{
						Status:     "ok",
						StatusCode: 200,
						Message:    "OK",
					},
				},
			},
			wantErr: false,
			checkRequest: func(t *testing.T, r *http.Request) {
				_ = r.ParseForm()
				// Optional fields should be empty
				if r.FormValue("comment") != "" {
					t.Errorf("Comment should be empty: %s", r.FormValue("comment"))
				}
				if r.FormValue("paymentModeId") != "" {
					t.Errorf("paymentmodeid should be empty: %s", r.FormValue("paymentModeId"))
				}
				if r.FormValue("categoryId") != "" {
					t.Errorf("categoryid should be empty: %s", r.FormValue("categoryId"))
				}
			},
		},
		{
			name: "server error",
			bill: Bill{
				What:    "Test",
				Amount:  10.00,
				PayerID: 1,
				OwedTo:  []int{1},
				Date:    "2024-01-15",
			},
			responseStatus: http.StatusInternalServerError,
			responseBody:   "Internal Server Error",
			wantErr:        true,
		},
		{
			name: "api error response",
			bill: Bill{
				What:    "Test",
				Amount:  10.00,
				PayerID: 1,
				OwedTo:  []int{1},
				Date:    "2024-01-15",
			},
			responseStatus: http.StatusOK,
			responseBody: OCSResponse{
				OCS: struct {
					Meta struct {
						Status     string `json:"status"`
						StatusCode int    `json:"statuscode"`
						Message    string `json:"message"`
					} `json:"meta"`
					Data json.RawMessage `json:"data"`
				}{
					Meta: struct {
						Status     string `json:"status"`
						StatusCode int    `json:"statuscode"`
						Message    string `json:"message"`
					}{
						Status:     "failure",
						StatusCode: 400,
						Message:    "Invalid bill data",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify common headers
				if r.Header.Get("OCS-APIRequest") != "true" {
					t.Error("Missing OCS-APIRequest header")
				}

				if tt.checkRequest != nil {
					tt.checkRequest(t, r)
				}

				w.WriteHeader(tt.responseStatus)
				if s, ok := tt.responseBody.(string); ok {
					_, _ = w.Write([]byte(s))
				} else {
					_ = json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			cfg := &config.Config{
				Domain:   server.URL,
				User:     "testuser",
				Password: "testpass",
			}
			client := NewClient(cfg)

			err := client.CreateBill("test-project", tt.bill)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateBill() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateBillWithCurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.FormValue("original_currency_id") != "5" {
			t.Errorf("Wrong original_currency_id: %s", r.FormValue("original_currency_id"))
		}

		response := OCSResponse{
			OCS: struct {
				Meta struct {
					Status     string `json:"status"`
					StatusCode int    `json:"statuscode"`
					Message    string `json:"message"`
				} `json:"meta"`
				Data json.RawMessage `json:"data"`
			}{
				Meta: struct {
					Status     string `json:"status"`
					StatusCode int    `json:"statuscode"`
					Message    string `json:"message"`
				}{
					Status:     "ok",
					StatusCode: 200,
					Message:    "OK",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{
		Domain:   server.URL,
		User:     "testuser",
		Password: "testpass",
	}
	client := NewClient(cfg)

	bill := Bill{
		What:               "Currency test",
		Amount:             100.00,
		PayerID:            1,
		OwedTo:             []int{1},
		Date:               "2024-01-15",
		OriginalCurrencyID: 5,
	}

	err := client.CreateBill("test-project", bill)
	if err != nil {
		t.Errorf("CreateBill() unexpected error: %v", err)
	}
}

func TestGetUserInfo(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   any
		wantErr        bool
		wantLocale     string
		wantLanguage   string
	}{
		{
			name:           "successful request",
			responseStatus: http.StatusOK,
			responseBody: OCSResponse{
				OCS: struct {
					Meta struct {
						Status     string `json:"status"`
						StatusCode int    `json:"statuscode"`
						Message    string `json:"message"`
					} `json:"meta"`
					Data json.RawMessage `json:"data"`
				}{
					Meta: struct {
						Status     string `json:"status"`
						StatusCode int    `json:"statuscode"`
						Message    string `json:"message"`
					}{
						Status:     "ok",
						StatusCode: 200,
						Message:    "OK",
					},
					Data: mustMarshal(map[string]string{"locale": "he_IL", "language": "he"}),
				},
			},
			wantErr:      false,
			wantLocale:   "he_IL",
			wantLanguage: "he",
		},
		{
			name:           "server error",
			responseStatus: http.StatusInternalServerError,
			responseBody:   "Internal Server Error",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/ocs/v2.php/cloud/user" {
					t.Errorf("Wrong path: got %s, want /ocs/v2.php/cloud/user", r.URL.Path)
				}

				w.WriteHeader(tt.responseStatus)
				if s, ok := tt.responseBody.(string); ok {
					_, _ = w.Write([]byte(s))
				} else {
					_ = json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			cfg := &config.Config{
				Domain:   server.URL,
				User:     "testuser",
				Password: "testpass",
			}
			client := NewClient(cfg)

			info, err := client.GetUserInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && info != nil {
				if info.Locale != tt.wantLocale {
					t.Errorf("GetUserInfo() Locale = %v, want %v", info.Locale, tt.wantLocale)
				}
				if info.Language != tt.wantLanguage {
					t.Errorf("GetUserInfo() Language = %v, want %v", info.Language, tt.wantLanguage)
				}
			}
		})
	}
}

func TestProjectCurrencyName(t *testing.T) {
	projectJSON := `{
		"id": "test",
		"name": "Test",
		"currencyname": "€",
		"members": [],
		"currencies": []
	}`

	var project Project
	if err := json.Unmarshal([]byte(projectJSON), &project); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if project.CurrencyName != "€" {
		t.Errorf("CurrencyName = %q, want %q", project.CurrencyName, "€")
	}

	// Test round-trip through marshal
	data, err := json.Marshal(project)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var project2 Project
	if err := json.Unmarshal(data, &project2); err != nil {
		t.Fatalf("Unmarshal round-trip error: %v", err)
	}

	if project2.CurrencyName != "€" {
		t.Errorf("Round-trip CurrencyName = %q, want %q", project2.CurrencyName, "€")
	}
}

func TestProjectUnmarshalObjectKeyed(t *testing.T) {
	// Real API returns categories and payment modes as objects keyed by ID
	projectJSON := `{
		"id": "test",
		"name": "Test",
		"currencyname": "₪",
		"members": [],
		"currencies": [],
		"categories": {
			"5": {"name": "Food", "icon": "🍔", "color": "#ff0000"},
			"12": {"name": "Transport", "icon": "🚗", "color": "#00ff00"}
		},
		"paymentmodes": {
			"3": {"name": "Credit Card", "icon": "💳", "color": "#0000ff"},
			"7": {"name": "Cash", "icon": "💵", "color": "#00ff00"}
		}
	}`

	var project Project
	if err := json.Unmarshal([]byte(projectJSON), &project); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Verify categories have correct IDs, icons, and colors from map keys
	catByName := make(map[string]Category)
	for _, c := range project.Categories {
		catByName[c.Name] = c
	}
	if catByName["Food"].ID != 5 {
		t.Errorf("Category Food ID = %d, want 5", catByName["Food"].ID)
	}
	if catByName["Food"].Icon != "\U0001F354" {
		t.Errorf("Category Food Icon = %q, want %q", catByName["Food"].Icon, "\U0001F354")
	}
	if catByName["Food"].Color != "#ff0000" {
		t.Errorf("Category Food Color = %q, want %q", catByName["Food"].Color, "#ff0000")
	}
	if catByName["Transport"].ID != 12 {
		t.Errorf("Category Transport ID = %d, want 12", catByName["Transport"].ID)
	}
	if catByName["Transport"].Icon != "\U0001F697" {
		t.Errorf("Category Transport Icon = %q, want %q", catByName["Transport"].Icon, "\U0001F697")
	}
	if catByName["Transport"].Color != "#00ff00" {
		t.Errorf("Category Transport Color = %q, want %q", catByName["Transport"].Color, "#00ff00")
	}

	// Verify payment modes have correct IDs, icons, and colors from map keys
	pmByName := make(map[string]PaymentMode)
	for _, pm := range project.PaymentModes {
		pmByName[pm.Name] = pm
	}
	if pmByName["Credit Card"].ID != 3 {
		t.Errorf("PaymentMode Credit Card ID = %d, want 3", pmByName["Credit Card"].ID)
	}
	if pmByName["Credit Card"].Icon != "\U0001F4B3" {
		t.Errorf("PaymentMode Credit Card Icon = %q, want %q", pmByName["Credit Card"].Icon, "\U0001F4B3")
	}
	if pmByName["Credit Card"].Color != "#0000ff" {
		t.Errorf("PaymentMode Credit Card Color = %q, want %q", pmByName["Credit Card"].Color, "#0000ff")
	}
	if pmByName["Cash"].ID != 7 {
		t.Errorf("PaymentMode Cash ID = %d, want 7", pmByName["Cash"].ID)
	}
	if pmByName["Cash"].Icon != "\U0001F4B5" {
		t.Errorf("PaymentMode Cash Icon = %q, want %q", pmByName["Cash"].Icon, "\U0001F4B5")
	}
	if pmByName["Cash"].Color != "#00ff00" {
		t.Errorf("PaymentMode Cash Color = %q, want %q", pmByName["Cash"].Color, "#00ff00")
	}
}

func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
