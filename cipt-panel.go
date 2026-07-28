package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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

func activateLicenseOnline(ctx context.Context, client *http.Client, baseURL Url, accessToken string, license string) error {

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

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearerToken(accessToken))

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

func issueRequestCertificate(ctx context.Context, client *http.Client, baseURL Url, accessToken string, device *Device, shopper *Consumer, path string) ([]byte, error) {
	rawUrl := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/security/openvpn_cert_req")

	fullUrl, err := url.Parse(rawUrl)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	query := fullUrl.Query()
	query.Set("common_name", device.CommonName)
	query.Set("org_name", shopper.OrgName)
	fullUrl.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fullUrl.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", bearerToken(accessToken))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус до чтения тела
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Читаем тело только для ошибки
		errorBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return nil, fmt.Errorf(
			"remote transaction failed: status=%s, body=%s",
			resp.Status,
			strings.TrimSpace(string(errorBody)),
		)
	}

	// Создаем файл
	filePath := fmt.Sprintf("%s/%s.csr", path, device.CommonName)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	// Копируем тело ответа напрямую в файл
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("save file: %w", err)
	}

	// Если нужно вернуть содержимое как []byte, прочитаем его из файла
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read saved file: %w", err)
	}

	return content, nil
}

//

func readCertsCommonName(dir string, opensslPath string) (map[string]string, error) {
	// Проверяем openssl один раз, чтобы его отсутствие не выглядело
	// как директория без сертификатов
	if _, err := exec.LookPath(opensslPath); err != nil {
		return nil, fmt.Errorf("openssl недоступен: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %q: %w", dir, err)
	}

	certs := make(map[string]string)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		commonName, err := readCommonName(path, opensslPath)
		if err != nil {
			fmt.Printf("Пропускаю %s: %v\n", entry.Name(), err)
			continue
		}

		certs[entry.Name()] = commonName
	}

	if len(certs) == 0 {
		return nil, fmt.Errorf("в директории %q не нашлось ни одного читаемого сертификата", dir)
	}

	return certs, nil
}

// Вытащить CommonName из одного сертификата через openssl
func readCommonName(path string, opensslPath string) (string, error) {
	// sep_multiline печатает каждое поле subject на своей строке: "    CN=имя"
	cmd := exec.Command(
		opensslPath, "x509",
		"-in", path,
		"-noout",
		"-subject",
		"-nameopt", "sep_multiline",
	)

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("openssl: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("запуск openssl: %w", err)
	}

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if commonName, ok := strings.CutPrefix(line, "CN="); ok {
			return commonName, nil
		}
	}

	return "", fmt.Errorf("в сертификате нет поля CN")
}

// Найти сертификат с нужным CommonName и передать его дальше
func findCertByCommonName(certs map[string]string, commonName string) (string, error) {
	for fileName, certCommonName := range certs {
		if certCommonName != commonName {
			continue
		}

		fmt.Printf("Найден сертификат %s с CommonName %s\n", fileName, certCommonName)
		return fileName, nil
	}

	return "", fmt.Errorf("сертификат с CommonName %q не найден", commonName)
}

func uploadClientCertificate(ctx context.Context, client *http.Client, baseURL Url, accessToken string, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file %q: %w", filePath, err)
	}
	defer file.Close()

	var requestBody bytes.Buffer

	writer := multipart.NewWriter(&requestBody)

	filePart, err := writer.CreateFormFile(
		"file",
		filepath.Base(filePath),
	)
	if err != nil {
		return fmt.Errorf("create multipart file field: %w", err)
	}

	if _, err := io.Copy(filePart, file); err != nil {
		return fmt.Errorf("copy file into multipart request: %w", err)
	}

	// Обязательно закрываем writer до отправки запроса.
	// Это добавляет завершающую multipart-границу.
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/security/openvpn_cert")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		&requestBody,
	)
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", bearerToken(accessToken))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send upload request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read upload response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"standby asset upload failed: status=%s, body=%s",
			resp.Status,
			strings.TrimSpace(string(responseBody)),
		)
	}

	return nil
}

func uploadCaCertificate(ctx context.Context, client *http.Client, baseURL Url, accessToken string, filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", filePath, err)
	}
	defer file.Close()

	var requestBody bytes.Buffer

	writer := multipart.NewWriter(&requestBody)

	if err := writer.WriteField("is_ca", "true"); err != nil {
		return nil, fmt.Errorf("write is_ca field: %w", err)
	}

	filePart, err := writer.CreateFormFile(
		"file",
		filepath.Base(filePath),
	)
	if err != nil {
		return nil, fmt.Errorf("create multipart file field: %w", err)
	}

	if _, err := io.Copy(filePart, file); err != nil {
		return nil, fmt.Errorf("copy file into multipart request: %w", err)
	}

	// Обязательно закрываем writer до отправки запроса.
	// Это добавляет завершающую multipart-границу.
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/security/openvpn_cert")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		&requestBody,
	)
	if err != nil {
		return nil, fmt.Errorf("create upload request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", bearerToken(accessToken))
	req.Header.Set("Content-Type", writer.FormDataContentType())

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
			"standby asset upload failed: status=%s, body=%s",
			resp.Status,
			strings.TrimSpace(string(responseBody)),
		)
	}

	return responseBody, nil
}

func startOpenVpn(ctx context.Context, client *http.Client, baseURL Url, accessToken string) ([]byte, error) {

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/security/openvpn_start")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

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

func setStunnelParametrs(ctx context.Context, client *http.Client, baseURL Url, accessToken string, payload StunnelParametrs) ([]byte, error) {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/security/cryptotunnel_conf")
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

func startStunnel(ctx context.Context, client *http.Client, baseURL Url, accessToken string) ([]byte, error) {

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/security/cryptotunnel_start")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

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
