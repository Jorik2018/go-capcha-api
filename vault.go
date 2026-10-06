package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type VaultResponse struct {
	Data VaultData `json:"data"`
}

type VaultData struct {
	Data map[string]interface{} `json:"data"`
}

// getVaultSecret obtiene una clave desde Vault.
//
// Si Vault responde correctamente y la clave existe:
//   - retorna el valor.
//
// Si ocurre cualquier error:
//   - si defaultValue != nil, retorna el valor por defecto.
//   - si defaultValue == nil, termina la aplicación.
func getVaultSecret(
	vaultURI string,
	vaultToken string,
	key string,
	defaultValue *string,
) string {

	fallback := func(err error) string {
		if defaultValue != nil {
			log.Printf(
				"WARNING: Could not get %s from Vault: %v. Using default value.",
				key,
				err,
			)

			return *defaultValue
		}

		log.Fatalf(
			"Could not get %s from Vault: %v",
			key,
			err,
		)

		return "" // log.Fatalf termina el proceso
	}

	// Validar configuración
	if vaultURI == "" {
		return fallback(
			fmt.Errorf("VAULT_URI is empty"),
		)
	}

	if vaultToken == "" {
		return fallback(
			fmt.Errorf("VAULT_TOKEN is empty"),
		)
	}

	// Crear request
	req, err := http.NewRequest(
		http.MethodGet,
		vaultURI,
		nil,
	)
	if err != nil {
		return fallback(err)
	}

	req.Header.Set(
		"X-Vault-Token",
		vaultToken,
	)

	// Ejecutar request
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return fallback(err)
	}
	defer response.Body.Close()

	// Leer respuesta
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fallback(err)
	}

	// Validar HTTP status
	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {

		return fallback(
			fmt.Errorf(
				"vault request failed: status=%d body=%s",
				response.StatusCode,
				string(body),
			),
		)
	}

	// Parsear JSON
	var vaultResponse VaultResponse

	if err := json.Unmarshal(
		body,
		&vaultResponse,
	); err != nil {
		return fallback(err)
	}

	// Buscar key
	value, exists := vaultResponse.Data.Data[key]

	if !exists {
		return fallback(
			fmt.Errorf(
				"secret key %q not found",
				key,
			),
		)
	}

	// null
	if value == nil {
		return fallback(
			fmt.Errorf(
				"secret key %q is null",
				key,
			),
		)
	}

	// Esperamos string
	result, ok := value.(string)

	if !ok {
		return fallback(
			fmt.Errorf(
				"secret key %q is not a string",
				key,
			),
		)
	}

	return result
}
