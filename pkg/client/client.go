package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type VaultClientConfig struct {
	Address string
}

type HealthResponse struct {
	Active        bool
	Standby       bool
	Uninitialized bool
	Sealed        bool
	StatusCode    int
}

type InitRequest struct {
	SecretShares    int `json:"secret_shares"`
	SecretThreshold int `json:"secret_threshold"`
}

type InitResponse struct {
	Keys       []string `json:"keys"`
	KeysBase64 []string `json:"keys_base64"`
	RootToken  string   `json:"root_token"`
}

type UnsealRequest struct {
	Key   string `json:"key"`
	Reset bool   `json:"reset"`
}

type UnsealResponse struct {
	Sealed   bool `json:"sealed"`
	T        int  `json:"t"`
	N        int  `json:"n"`
	Progress int  `json:"progress"`
}

type UnsealEvent struct {
	Sealed       bool
	KeysProvided int
	KeysRequired int
}

var (
	httpClientLock = &sync.Mutex{}
	httpClient     http.Client
)

func initClient() http.Client {
	if &httpClient == nil {
		httpClientLock.Lock()
		defer httpClientLock.Unlock()
		if &httpClient == nil {
			httpClient = http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
		}
	}
	return httpClient
}

func Health(config VaultClientConfig) (*HealthResponse, error) {
	endpoint := fmt.Sprintf("%v/v1/sys/health", config.Address)

	httpClient = initClient()

	response, err := httpClient.Head(endpoint)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case 200:
		return &HealthResponse{StatusCode: response.StatusCode, Active: true}, nil
	case 429:
		return &HealthResponse{StatusCode: response.StatusCode, Standby: true}, nil
	case 501:
		return &HealthResponse{StatusCode: response.StatusCode, Uninitialized: true}, nil
	case 503:
		return &HealthResponse{StatusCode: response.StatusCode, Sealed: true}, nil
	default:
		return &HealthResponse{StatusCode: response.StatusCode}, nil
	}
}

func Initialize(config VaultClientConfig) (*InitResponse, error) {
	endpoint := fmt.Sprintf("%v/v1/sys/init", config.Address)
	request := InitRequest{
		SecretShares:    5,
		SecretThreshold: 3,
	}

	var response InitResponse
	if err := vaultRequest[InitRequest, *InitResponse](http.MethodPut, endpoint, request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func Unseal(config VaultClientConfig, key string) (*UnsealEvent, error) {
	endpoint := fmt.Sprintf("%v/v1/sys/unseal", config.Address)
	request := UnsealRequest{
		Key: key,
	}

	var response UnsealResponse
	if err := vaultRequest[UnsealRequest, *UnsealResponse](http.MethodPut, endpoint, request, &response); err != nil {
		return nil, err
	}

	target := response.T
	var progress int
	if response.Sealed {
		progress = response.Progress
	} else {
		progress = response.T
	}

	return &UnsealEvent{
		Sealed:       response.Sealed,
		KeysProvided: progress,
		KeysRequired: target,
	}, nil
}

func vaultRequest[K any, V any](method string, endpoint string, body K, response V) error {
	requestData, _ := json.Marshal(&body)
	requestBytes := bytes.NewReader(requestData)

	request, err := http.NewRequest(method, endpoint, requestBytes)
	if err != nil {
		return fmt.Errorf("Error creating request: %w", err)
	}

	httpClient = initClient()

	httpResponse, err := httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("Response error: %w", err)
	}
	defer httpResponse.Body.Close()

	httpResponseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return fmt.Errorf("Error reading Vault response: %w", err)
	}

	if httpResponse.StatusCode != 200 {
		return fmt.Errorf("Vault operation failed [%d]: %w", httpResponse.StatusCode, err)
	}

	if err := json.Unmarshal(httpResponseBody, &response); err != nil {
		return fmt.Errorf("Failed to read Vault response: %w", err)
	}

	return nil
}
