package invoice_repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dnjooiopa/phone-charging-locker/internal/usecase"
)

type phoenixdClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewPhoenixd(baseURL, apiKey string) usecase.InvoiceRepository {
	return &phoenixdClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type createInvoiceResponse struct {
	AmountSat   int64  `json:"amountSat"`
	PaymentHash string `json:"paymentHash"`
	Serialized  string `json:"serialized"`
}

func (c *phoenixdClient) CreateInvoice(ctx context.Context, params *usecase.CreateInvoiceParams) (*usecase.CreateInvoiceResult, error) {
	form := url.Values{}
	form.Set("description", params.Description)
	form.Set("amountSat", fmt.Sprintf("%d", params.AmountSat))
	if params.ExternalID != "" {
		form.Set("externalId", params.ExternalID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/createinvoice", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-API-KEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("phoenixd proxy returned status %d: %s", resp.StatusCode, string(body))
	}

	var result createInvoiceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &usecase.CreateInvoiceResult{
		PaymentHash: result.PaymentHash,
		Serialized:  result.Serialized,
	}, nil
}
