package goupc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const baseURL = "https://go-upc.com/api/v1/code/"

// Client calls the go-upc.com API.
// The API key is passed as a query parameter named "key".
type Client struct {
	apiKey     string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// UPCResponse is the top-level response from go-upc.com.
type UPCResponse struct {
	Code       string     `json:"code"`
	CodeType   string     `json:"codeType"`
	Product    UPCProduct `json:"product"`
	BarcodeURL string     `json:"barcodeUrl"`
	Inferred   bool       `json:"inferred"`
}

// UPCProduct holds the product details within the response.
type UPCProduct struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Brand        string `json:"brand"`
	ImageURL     string `json:"imageUrl"`
	Region       string `json:"region"`
	Category     string `json:"category"`
}

// Lookup fetches product data for a barcode from go-upc.com.
// Returns ErrNotFound if the barcode is unknown to go-upc.com.
func (c *Client) Lookup(ctx context.Context, upc string) (UPCResponse, error) {
	u, err := url.Parse(baseURL + url.PathEscape(upc))
	if err != nil {
		return UPCResponse{}, fmt.Errorf("building URL: %w", err)
	}
	q := u.Query()
	q.Set("key", c.apiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return UPCResponse{}, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "thoth-collection-manager/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return UPCResponse{}, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return UPCResponse{}, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return UPCResponse{}, fmt.Errorf("go-upc.com returned HTTP %d", resp.StatusCode)
	}

	var result UPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return UPCResponse{}, fmt.Errorf("decoding response: %w", err)
	}
	return result, nil
}
