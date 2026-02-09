package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/chenasraf/cospend-cli/internal/api"
)

const (
	cacheTTL = 1 * time.Hour
	appName  = "cospend"
)

// currencyCodeToSymbol maps currency codes to their symbols
var currencyCodeToSymbol = map[string]string{
	"aed": "د.إ",
	"afn": "؋",
	"all": "Lek",
	"amd": "դր.",
	"ars": "$",
	"aud": "$",
	"azn": "ман.",
	"bam": "KM",
	"bdt": "৳",
	"bgn": "лв.",
	"bhd": "د.ب.",
	"bif": "FBu",
	"bnd": "$",
	"bob": "Bs",
	"brl": "R$",
	"bwp": "P",
	"byn": "руб.",
	"bzd": "$",
	"cad": "$",
	"cdf": "FrCD",
	"chf": "CHF",
	"clp": "$",
	"cny": "¥",
	"cop": "$",
	"crc": "₡",
	"cup": "$",
	"cve": "CV$",
	"czk": "Kč",
	"djf": "Fdj",
	"dkk": "kr",
	"dop": "RD$",
	"dzd": "د.ج.",
	"egp": "ج.م.",
	"etb": "Br",
	"eur": "€",
	"gbp": "£",
	"gel": "GEL",
	"ghs": "GH₵",
	"gnf": "FG",
	"gtq": "Q",
	"hkd": "$",
	"hnl": "L",
	"huf": "Ft",
	"idr": "Rp",
	"ils": "₪",
	"inr": "₹",
	"iqd": "د.ع.",
	"irr": "﷼",
	"isk": "kr",
	"jmd": "$",
	"jod": "د.أ.",
	"jpy": "¥",
	"kes": "Ksh",
	"khr": "៛",
	"kmf": "FC",
	"krw": "₩",
	"kwd": "د.ك.",
	"kzt": "тңг.",
	"lbp": "ل.ل.",
	"lkr": "Rs",
	"lyd": "د.ل.",
	"mad": "د.م.",
	"mdl": "MDL",
	"mga": "MGA",
	"mkd": "MKD",
	"mmk": "K",
	"mop": "MOP$",
	"mur": "MURs",
	"mxn": "$",
	"myr": "RM",
	"mzn": "MTn",
	"nad": "N$",
	"ngn": "₦",
	"nio": "C$",
	"nok": "kr",
	"npr": "Rs",
	"nzd": "$",
	"omr": "ر.ع.",
	"pab": "B/.",
	"pen": "S/.",
	"php": "₱",
	"pkr": "₨",
	"pln": "zł",
	"pyg": "₲",
	"qar": "ر.ق.",
	"ron": "RON",
	"rsd": "дин.",
	"rub": "₽",
	"rwf": "FR",
	"sar": "﷼",
	"sdg": "SDG",
	"sek": "kr",
	"sgd": "$",
	"sos": "Ssh",
	"thb": "฿",
	"tnd": "د.ت.",
	"top": "T$",
	"try": "₺",
	"ttd": "$",
	"twd": "NT$",
	"tzs": "TSh",
	"uah": "₴",
	"ugx": "USh",
	"usd": "$",
	"uyu": "$",
	"uzs": "UZS",
	"vnd": "₫",
	"xaf": "FCFA",
	"xcd": "EC$",
	"xof": "CFA",
	"yer": "ر.ي.",
	"zar": "R",
}

// CachedProject stores project data with timestamp
type CachedProject struct {
	Project  *api.Project `json:"project"`
	CachedAt time.Time    `json:"cached_at"`
}

// getCacheHome returns the cache home directory, checking XDG_CACHE_HOME env var first
func getCacheHome() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return dir
	}
	return xdg.CacheHome
}

// getCachePath returns the cache file path for a project
func getCachePath(projectID string) (string, error) {
	cacheDir := filepath.Join(getCacheHome(), appName)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("creating cache directory: %w", err)
	}
	return filepath.Join(cacheDir, fmt.Sprintf("%s.json", projectID)), nil
}

// Load retrieves cached project data if it exists and is not expired
func Load(projectID string) (*api.Project, bool) {
	path, err := getCachePath(projectID)
	if err != nil {
		return nil, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var cached CachedProject
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}

	// Check if cache is expired
	if time.Since(cached.CachedAt) > cacheTTL {
		return nil, false
	}

	return cached.Project, true
}

// Save stores project data in the cache
func Save(projectID string, project *api.Project) error {
	path, err := getCachePath(projectID)
	if err != nil {
		return err
	}

	cached := CachedProject{
		Project:  project,
		CachedAt: time.Now(),
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache data: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}

	return nil
}

// CachedUserInfo stores user info data with timestamp
type CachedUserInfo struct {
	UserInfo *api.UserInfo `json:"user_info"`
	CachedAt time.Time     `json:"cached_at"`
}

// LoadUserInfo retrieves cached user info if it exists and is not expired
func LoadUserInfo() (*api.UserInfo, bool) {
	cacheDir := filepath.Join(getCacheHome(), appName)
	path := filepath.Join(cacheDir, "_userinfo.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var cached CachedUserInfo
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false
	}

	if time.Since(cached.CachedAt) > cacheTTL {
		return nil, false
	}

	return cached.UserInfo, true
}

// SaveUserInfo stores user info in the cache
func SaveUserInfo(userInfo *api.UserInfo) error {
	cacheDir := filepath.Join(getCacheHome(), appName)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	path := filepath.Join(cacheDir, "_userinfo.json")

	cached := CachedUserInfo{
		UserInfo: userInfo,
		CachedAt: time.Now(),
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling user info cache: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing user info cache: %w", err)
	}

	return nil
}

var (
	symbolToISOMap  map[string]string
	symbolToISOOnce sync.Once
)

// SymbolToISO returns the uppercase ISO currency code for a given symbol.
// For ambiguous symbols (e.g. "$"), it prefers USD.
func SymbolToISO(symbol string) string {
	symbolToISOOnce.Do(func() {
		// Preferred codes for ambiguous symbols - first match wins
		preferred := []string{"usd", "cny", "gbp", "eur"}
		symbolToISOMap = make(map[string]string)

		// First pass: set all mappings (last write wins)
		for code, sym := range currencyCodeToSymbol {
			symbolToISOMap[sym] = strings.ToUpper(code)
		}
		// Second pass: override with preferred codes for ambiguous symbols
		for _, code := range preferred {
			if sym, ok := currencyCodeToSymbol[code]; ok {
				symbolToISOMap[sym] = strings.ToUpper(code)
			}
		}
	})

	if iso, ok := symbolToISOMap[symbol]; ok {
		return iso
	}
	return ""
}

// ResolveMember finds a member by username (case-insensitive) and returns their ID
func ResolveMember(project *api.Project, username string) (int, error) {
	lowerUsername := strings.ToLower(username)
	for _, m := range project.Members {
		if strings.ToLower(m.Name) == lowerUsername || strings.ToLower(m.UserID) == lowerUsername {
			return m.ID, nil
		}
	}
	return 0, fmt.Errorf("member not found: %s", username)
}

// ResolveCategory finds a category by name (case-insensitive, substring) or ID and returns the ID
func ResolveCategory(project *api.Project, nameOrID string) (int, error) {
	if nameOrID == "" {
		return 0, fmt.Errorf("category not found: %s", nameOrID)
	}

	// Try parsing as ID first
	if id, err := strconv.Atoi(nameOrID); err == nil {
		for _, c := range project.Categories {
			if c.ID == id {
				return id, nil
			}
		}
	}

	lowerName := strings.ToLower(nameOrID)

	// Try exact match first
	for _, c := range project.Categories {
		if strings.ToLower(c.Name) == lowerName {
			return c.ID, nil
		}
	}

	// Fallback to substring match
	for _, c := range project.Categories {
		if strings.Contains(strings.ToLower(c.Name), lowerName) {
			return c.ID, nil
		}
	}

	return 0, fmt.Errorf("category not found: %s", nameOrID)
}

// ResolvePaymentMode finds a payment mode by name (case-insensitive, substring) or ID and returns the ID
func ResolvePaymentMode(project *api.Project, nameOrID string) (int, error) {
	if nameOrID == "" {
		return 0, fmt.Errorf("payment mode not found: %s", nameOrID)
	}

	// Try parsing as ID first
	if id, err := strconv.Atoi(nameOrID); err == nil {
		for _, pm := range project.PaymentModes {
			if pm.ID == id {
				return id, nil
			}
		}
	}

	lowerName := strings.ToLower(nameOrID)

	// Try exact match first
	for _, pm := range project.PaymentModes {
		if strings.ToLower(pm.Name) == lowerName {
			return pm.ID, nil
		}
	}

	// Fallback to substring match
	for _, pm := range project.PaymentModes {
		if strings.Contains(strings.ToLower(pm.Name), lowerName) {
			return pm.ID, nil
		}
	}

	return 0, fmt.Errorf("payment mode not found: %s", nameOrID)
}

// ResolveCurrency finds a currency by name (case-insensitive), ID, or currency code symbol and returns the currency
func ResolveCurrency(project *api.Project, nameOrID string) (*api.Currency, error) {
	// Try parsing as ID first
	if id, err := strconv.Atoi(nameOrID); err == nil {
		for i := range project.Currencies {
			if project.Currencies[i].ID == id {
				return &project.Currencies[i], nil
			}
		}
	}

	// Try matching by name (case-insensitive)
	lowerName := strings.ToLower(nameOrID)
	for i := range project.Currencies {
		if strings.ToLower(project.Currencies[i].Name) == lowerName {
			return &project.Currencies[i], nil
		}
	}

	// Try matching by currency code symbol (e.g., "usd" -> "$")
	if symbol, ok := currencyCodeToSymbol[lowerName]; ok {
		for i := range project.Currencies {
			if strings.Contains(project.Currencies[i].Name, symbol) {
				return &project.Currencies[i], nil
			}
		}
	}

	return nil, fmt.Errorf("currency not found: %s", nameOrID)
}
