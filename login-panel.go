package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func login(ctx context.Context, client *http.Client, baseURL Url, username, password string) (*LoginResponse, error) {
	requestBody := LoginRequest{
		Username: username,
		Password: password,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/auth/login")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Ограничиваем размер ответа, чтобы не прочитать
	// бесконечно большой ответ в память.
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"login failed: status=%s, body=%s",
			resp.Status,
			strings.TrimSpace(string(responseBody)),
		)
	}

	var result LoginResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf(
			"decode response: %w; body=%s",
			err,
			strings.TrimSpace(string(responseBody)),
		)
	}

	if result.AccessToken == "" {
		return nil, fmt.Errorf("response does not contain access_token")
	}

	return &result, nil
}

func bearerToken(token string) string {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return token
	}
	return "Bearer " + token
}

func refreshTokens(ctx context.Context, client *http.Client, baseURL Url, tokens *LoginResponse) error {

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/auth/refresh")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return fmt.Errorf("create refresh request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", bearerToken(tokens.RefreshToken))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send refresh request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read refresh response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"token refresh failed: status=%s, body=%s",
			resp.Status,
			strings.TrimSpace(string(responseBody)),
		)
	}

	if err := json.Unmarshal(responseBody, tokens); err != nil {
		return fmt.Errorf(
			"decode refresh response: %w; body=%s",
			err,
			strings.TrimSpace(string(responseBody)),
		)
	}

	if tokens.AccessToken == "" {
		return fmt.Errorf("refresh response does not contain access_token")
	}

	return nil
}
