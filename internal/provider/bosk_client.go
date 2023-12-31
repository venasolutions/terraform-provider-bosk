package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type BoskClient struct {
	httpClient *http.Client
	urlPrefix  string
	auth       *BasicAuth
}

type BasicAuth struct {
	username string
	password string
}

func NewBoskClientWithoutAuth(httpClient *http.Client, urlPrefix string) *BoskClient {
	return &BoskClient{
		httpClient: httpClient,
		urlPrefix:  urlPrefix,
		auth:       nil,
	}
}

func NewBoskClient(httpClient *http.Client, urlPrefix string, username string, password string) *BoskClient {
	return &BoskClient{
		httpClient: httpClient,
		urlPrefix:  urlPrefix,
		auth: &BasicAuth{
			username: username,
			password: password,
		},
	}
}

// Portions taken from: https://github.com/hashicorp/terraform-provider-http/blob/main/internal/provider/data_source_http.go
func (client *BoskClient) GetJSONAsString(url string, diag *diag.Diagnostics) string {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to create HTTP request: %s", err))
		return "ERROR"
	}

	if client.auth != nil {
		req.SetBasicAuth(client.auth.username, client.auth.password)
	}
	httpResp, err := client.httpClient.Do(req)
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to %v %v: %s", req.Method, url, err))
		return "ERROR"
	}

	defer httpResp.Body.Close()

	if httpResp.StatusCode/100 != 2 {
		diag.AddError("Client Error", fmt.Sprintf("%v %v returned unexpected status %s", req.Method, url, httpResp.Status))
		return "ERROR"
	}

	bytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		diag.AddError(
			"Error reading response body",
			fmt.Sprintf("Error reading response body: %s", err),
		)
		return "ERROR"
	}
	if !utf8.Valid(bytes) {
		diag.AddWarning(
			"Response body is not recognized as UTF-8",
			"Terraform may not properly handle the response_body if the contents are binary.",
		)
	}

	normalized, err := normalizeJSON(bytes)
	if err != nil {
		diag.AddWarning(
			"Error normalizing JSON response",
			fmt.Sprintf("Error reading response body: %s", err),
		)
		return string(bytes)
	}

	return string(normalized)
}

func normalizeJSON(input []byte) ([]byte, error) {
	var parsed interface{}
	err := json.Unmarshal(input, &parsed)
	if err != nil {
		return input, err
	}
	result, err := json.Marshal(parsed)
	if err != nil {
		return input, err
	}
	return result, nil
}

func (client *BoskClient) PutJSONAsString(url string, value string, diag *diag.Diagnostics) {
	req, err := http.NewRequest("PUT", url, bytes.NewReader([]byte(value)))
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to create HTTP PUT request: %s", err))
		return
	}
	if client.auth != nil {
		req.SetBasicAuth(client.auth.username, client.auth.password)
	}

	httpResp, err := client.httpClient.Do(req)
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to PUT node: %s", err))
		return
	}

	defer httpResp.Body.Close()

	if httpResp.StatusCode/100 != 2 {
		diag.AddError("Client Error", fmt.Sprintf("PUT returned unexpected status: %s", httpResp.Status))
	}
}

func (client *BoskClient) Delete(url string, diag *diag.Diagnostics) {
	req, err := http.NewRequest("DELETE", url, nil)
	if client.auth != nil {
		req.SetBasicAuth(client.auth.username, client.auth.password)
	}
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to create HTTP DELETE request: %s", err))
		return
	}

	httpResp, err := client.httpClient.Do(req)
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to DELETE node: %s", err))
		return
	}

	defer httpResp.Body.Close()

	if httpResp.StatusCode/100 != 2 {
		diag.AddError("Client Error", fmt.Sprintf("DELETE returned unexpected status: %s", httpResp.Status))
	}
}
