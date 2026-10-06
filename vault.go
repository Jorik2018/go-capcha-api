package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type VaultResponse struct {
	Data VaultData `json:"data"`
}

type VaultData struct {
	Data map[string]interface{} `json:"data"`
}

func getVaultSecret(
	vaultURI string,
	vaultToken string,
	key string,
) (string, error) {

	req, err := http.NewRequest(
		http.MethodGet,
		vaultURI,
		nil,
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("X-Vault-Token", vaultToken)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf(
			"vault request failed: status=%d body=%s",
			response.StatusCode,
			string(body),
		)
	}

	var vaultResponse VaultResponse

	if err := json.Unmarshal(body, &vaultResponse); err != nil {
		return "", err
	}

	value, ok := vaultResponse.Data.Data[key]
	if !ok {
		return "", fmt.Errorf(
			"secret key %q not found",
			key,
		)
	}

	result, ok := value.(string)
	if !ok {
		return "", fmt.Errorf(
			"secret key %q is not a string",
			key,
		)
	}

	return result, nil
}
