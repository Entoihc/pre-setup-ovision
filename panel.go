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
	"os/exec"
	"path/filepath"
	"strings"
)

// Задать параметры для внешнего распознавания

// Настроить режим внешнего распознавания
func setRemoteTransactionParameters(ctx context.Context, client *http.Client, baseURL Url, accessToken string, payload RemoteTransactionRequest) ([]byte, error) {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/pipelineomini/remote_transaction")
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

func getInfoDevice(ctx context.Context, client *http.Client, baseURL Url, accessToken string, device *Device) error {
	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/get_hardware")
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", bearerToken(accessToken))

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
			"get hardware failed: status=%s, body=%s",
			resp.Status,
			strings.TrimSpace(string(responseBody)),
		)
	}

	if err = json.Unmarshal(responseBody, device); err != nil {
		return fmt.Errorf(
			"Ну удалось распарсить ответ от девайса с информацией о нем: %w; body=%s",
			err,
			strings.TrimSpace(string(responseBody)),
		)
	}

	return nil
}

func getDeviceName(shopper *Consumer, device *Device) error {

	check := false

	if shopper.Name == "t2" {
		device.Name = device.NumDote

		cutMAC := strings.ReplaceAll(device.Mac, ":", "")
		if len(cutMAC) < 6 {
			return fmt.Errorf("некорректный MAC-адрес: %s", device.Mac)
		}
		cutMAC = strings.ToUpper(cutMAC[len(cutMAC)-6:])
		device.CommonName = fmt.Sprintf("BT-%s-%s", cutMAC, device.NumDote)
		check = true
	}

	if shopper.Name == "ovision" {
		device.Name = device.Mac
		device.CommonName = device.Mac
		check = true
	}

	if !check {
		return (fmt.Errorf("Такого клиента не существует: %s", shopper.Name))
	}

	return nil
}

// Загрузить фото для standby режима
func uploadStandbyAsset(ctx context.Context, client *http.Client, baseURL Url, accessToken string, filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", filePath, err)
	}
	defer file.Close()

	var requestBody bytes.Buffer

	writer := multipart.NewWriter(&requestBody)

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

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/pipelineomini/installassets/standby")
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

// Загрузить фото для режима ожидания
func uploadWaitAsset(ctx context.Context, client *http.Client, baseURL Url, accessToken string, filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", filePath, err)
	}
	defer file.Close()

	var requestBody bytes.Buffer

	writer := multipart.NewWriter(&requestBody)

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

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/pipelineomini/installassets/waiting")
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
			"wait asset upload failed: status=%s, body=%s",
			resp.Status,
			strings.TrimSpace(string(responseBody)),
		)
	}

	return responseBody, nil
}

// Изменить параметры дисплея
func setDisplayParameters(ctx context.Context, client *http.Client, baseURL Url, accessToken string, payload DisplayParameters) ([]byte, error) {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	url := fmt.Sprintf("%s://%s:%s%s", baseURL.protocol, baseURL.host, baseURL.portPanel, "/pipelineomini/display")
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

// Инициализировать случайное число
func setInitSeed(ctx context.Context, client *http.Client, baseURL Url) error {
	cmd := exec.Command("ssh", "-i", "~/.ssh/id_rsa", "root@"+baseURL.host, "mkdir -p /root/.magprocryptopack && openssl rand -out /root/.magprocryptopack/random_seed 40")
	return cmd.Run()
}
