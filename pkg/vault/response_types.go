package vault

// JSON API types

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

// Custom client results

type UnsealState struct {
	Sealed       bool
	KeysProvided int
	KeysRequired int
}

type HealthState struct {
	Active        bool
	Standby       bool
	Uninitialized bool
	Sealed        bool
	StatusCode    int
}
