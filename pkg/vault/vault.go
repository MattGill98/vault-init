package vault

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Vault interface {
	HealthCheck() (*HealthResponse, error)
	Initialize() (*InitResponse, error)
	Unseal(string) (*UnsealEvent, error)
}

type vaultClient struct {
	address    string
	httpClient http.Client
}

func NewVaultClient(address string) Vault {
	return &vaultClient{
		address: address,
		httpClient: http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}
}

func (vaultClient *vaultClient) HealthCheck() (*HealthResponse, error) {
	endpoint := fmt.Sprintf("%v/v1/sys/health", vaultClient.address)

	response, err := vaultClient.httpClient.Head(endpoint)
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

func (vaultClient *vaultClient) Initialize() (*InitResponse, error) {
	endpoint := fmt.Sprintf("%v/v1/sys/init", vaultClient.address)
	request := InitRequest{
		SecretShares:    5,
		SecretThreshold: 3,
	}

	var response InitResponse
	if err := vaultRequest[InitRequest, *InitResponse](vaultClient, http.MethodPut, endpoint, request, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (vaultClient *vaultClient) Unseal(key string) (*UnsealEvent, error) {
	endpoint := fmt.Sprintf("%v/v1/sys/unseal", vaultClient.address)
	request := UnsealRequest{
		Key: key,
	}

	var response UnsealResponse
	if err := vaultRequest[UnsealRequest, *UnsealResponse](vaultClient, http.MethodPut, endpoint, request, &response); err != nil {
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

func vaultRequest[K any, V any](client *vaultClient, method string, endpoint string, body K, response V) error {
	requestData, _ := json.Marshal(&body)
	requestBytes := bytes.NewReader(requestData)

	request, err := http.NewRequest(method, endpoint, requestBytes)
	if err != nil {
		return fmt.Errorf("Error creating request: %w", err)
	}

	httpResponse, err := client.httpClient.Do(request)
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
