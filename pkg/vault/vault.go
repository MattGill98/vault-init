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
	HealthCheck() (HealthState, error)
	Initialize() (InitState, error)
	Unseal(string) (UnsealState, error)
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

func (vaultClient *vaultClient) HealthCheck() (HealthState, error) {
	endpoint := fmt.Sprintf("%v/v1/sys/health", vaultClient.address)

	response, err := vaultClient.httpClient.Head(endpoint)
	if err != nil {
		return HealthState{}, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case 200:
		return HealthState{StatusCode: response.StatusCode, Active: true}, nil
	case 429:
		return HealthState{StatusCode: response.StatusCode, Standby: true}, nil
	case 501:
		return HealthState{StatusCode: response.StatusCode, Uninitialized: true}, nil
	case 503:
		return HealthState{StatusCode: response.StatusCode, Sealed: true}, nil
	default:
		return HealthState{StatusCode: response.StatusCode}, nil
	}
}

func (vaultClient *vaultClient) Initialize() (InitState, error) {
	endpoint := fmt.Sprintf("%v/v1/sys/init", vaultClient.address)
	request := InitRequest{
		SecretShares:    5,
		SecretThreshold: 3,
	}

	var response InitResponse
	if err := vaultRequest[InitRequest, *InitResponse](vaultClient, http.MethodPut, endpoint, request, &response); err != nil {
		return InitState{}, err
	}

	return InitState{Keys: response.Keys, RootToken: response.RootToken}, nil
}

func (vaultClient *vaultClient) Unseal(key string) (UnsealState, error) {
	endpoint := fmt.Sprintf("%v/v1/sys/unseal", vaultClient.address)
	request := UnsealRequest{
		Key: key,
	}

	var response UnsealResponse
	if err := vaultRequest[UnsealRequest, *UnsealResponse](vaultClient, http.MethodPut, endpoint, request, &response); err != nil {
		return UnsealState{}, err
	}

	target := response.T
	var progress int
	if response.Sealed {
		progress = response.Progress
	} else {
		progress = response.T
	}

	return UnsealState{
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
