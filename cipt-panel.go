package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Загрузить СКЗИ через веб-панель
func uploadCipt(ctx context.Context, client *http.Client, baseURL Url, accessToken string, filePath string, cipt string) ([]byte, error) {

	installed, err := checkInstalledCipt(ctx, client, baseURL, cipt)

	if installed {
		return nil, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", filePath, err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Добавляем поле "details" с JSON
	detailsField, err := writer.CreateFormField("details")
	if err != nil {
		return nil, fmt.Errorf("create details field: %w", err)
	}
	_, err = detailsField.Write([]byte(`{"self_install":true}`))
	if err != nil {
		return nil, fmt.Errorf("write details: %w", err)
	}

	// Добавляем файл в поле "files[0]"
	filePart, err := writer.CreateFormFile(
		"files[0]",
		filepath.Base(filePath),
	)
	if err != nil {
		return nil, fmt.Errorf("create multipart file field: %w", err)
	}

	if _, err := io.Copy(filePart, file); err != nil {
		return nil, fmt.Errorf("copy file into multipart request: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/security/install_package")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		&requestBody,
	)
	if err != nil {
		return nil, fmt.Errorf("create upload request: %w", err)
	}

	// Заголовки как в curl
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:153.0) Gecko/20100101 Firefox/153.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Authorization", bearerToken(accessToken))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", fmt.Sprintf("%s://%s:%s", baseURL.protocol, baseURL.host, baseURL.portPanel))
	req.Header.Set("Referer", fmt.Sprintf("%s://%s:%s/", baseURL.protocol, baseURL.host, baseURL.portPanel))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send upload request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read upload response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"failed to upload CIPT: status=%s, body=%s",
			resp.Status,
			strings.TrimSpace(string(responseBody)),
		)
	}

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		installed, err := checkInstalledCipt(ctx, client, baseURL, cipt)
		if err != nil {
			return nil, fmt.Errorf(": %v\n", err)
		}
		if installed {
			fmt.Printf("Установлен СКЗИ: %s\n", cipt)
			time.Sleep(2 * time.Second)
			return responseBody, nil
		}
		time.Sleep(5 * time.Second)
	}

	return responseBody, fmt.Errorf("Ошибка при опросе статуса после установки СКЗИ")
}

func checkInstalledCipt(ctx context.Context, client *http.Client, baseURL Url, cipt string) (bool, error) {
	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/security/status")

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return false, fmt.Errorf("create check request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("send check request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false, fmt.Errorf("read check response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf(
			"check failed: status=%s, body=%s",
			resp.Status,
			strings.TrimSpace(string(responseBody)),
		)
	}

	var status securityStatus
	if err := json.Unmarshal(responseBody, &status); err != nil {
		return false, fmt.Errorf("decode check response: %w; body=%s", err, strings.TrimSpace(string(responseBody)))
	}

	var installed bool = false

	switch cipt {
	case "openvpn":
		if status.Data.OpenVPN_gost != "not_installed" {
			installed = true
		}
	case "openssl": // несколько значений через запятую
		if status.Data.OpenSSL != "not_installed" {
			installed = true
		}
	case "gmkseed": // несколько значений через запятую
		if status.Data.Cipf != "not_installed" {
			installed = true
		}
	case "stunnel": // несколько значений через запятую
		if status.Data.CryptoTunnel != "not_installed" {
			installed = true
		}
	case "license": // несколько значений через запятую
		if status.Data.LicenseActivated {
			installed = true
		}
	default:
		return false, fmt.Errorf("Ошибка проверки установленного СКЗИ, такого СКЗИ не существует: %s", cipt)
	}

	return installed, nil
}

func activateLicenseOnline(ctx context.Context, client *http.Client, baseURL Url, license string) error {

	installed, err := checkInstalledCipt(ctx, client, baseURL, "license")

	if installed {
		fmt.Println("Лицензия уже активирована")
		return nil
	}

	body, err := json.Marshal(struct {
		License string `json:"key"`
	}{License: license})
	if err != nil {
		return fmt.Errorf("encode request: %s", err)
	}

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/security/online_license")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create check request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send check request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read check response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"check failed: status=%s, body=%s",
			resp.Status,
			strings.TrimSpace(string(responseBody)),
		)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		installed, err := checkInstalledCipt(ctx, client, baseURL, "license")
		if err != nil {
			return fmt.Errorf(": %v\n", err)
		}
		if installed {
			fmt.Println("Лиценизия активирована")
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("Вышел таймаут проверки активации лицензии")
}

// Адрес сервера OpenVPN
type OpenVpnAddress struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

// Задать параметры OpenVPN
type OpenVpnParametrs struct {
	Addresses []OpenVpnAddress `json:"addresses"`
	TunMTU    int              `json:"tun_mtu"`
	Protocol  string           `json:"protocol"`
	CrlVerify bool             `json:"crl_verify"`
}

func setOpenVpnParametrs(ctx context.Context, client *http.Client, baseURL Url, accessToken string, payload OpenVpnParametrs) ([]byte, error) {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/security/openvpn_conf")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerToken(accessToken))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"remote transaction failed: status=%s, body=%s",
			resp.Status,
			strings.TrimSpace(string(responseBody)),
		)
	}

	return responseBody, nil
}
