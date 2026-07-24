package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"mime/multipart"
	"os"
	"path/filepath"
)


// Задать параметры для внешнего распознавания 
type RemoteTransactionRequest struct {
	Enabled              bool   `json:"enabled"`
	DeviceName           string `json:"deviceName"`
	DeviceNameIsHostName bool   `json:"deviceNameIsHostName"`
	PingURL              string `json:"pingUrl"`
	TimePing             int    `json:"timePing"`
}

//Настроить режим внешнего распознавания
func sendRemoteTransaction(ctx context.Context, client *http.Client, baseURL string, accessToken string, payload RemoteTransactionRequest) ([]byte, error) {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	url := fmt.Sprintf("%s%s", baseURL, "/pipelineomini/remote_transaction")
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

func getInfoDevice(ctx context.Context, client *http.Client, baseURL string, accessToken string, device *Device) error {
	url := fmt.Sprintf("%s%s", baseURL, "/get_hardware" )
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
			"parse get hardware response: %w; body=%s", 
			err, 
			strings.TrimSpace(string(responseBody)), 
		)
	}

	return nil
}

func getDeviceName(shopper Consumer, device *Device) error {
	
	check := false 
	
	if shopper.Name == "t2" {
		device.Name = shopper.NumDote

		cutMAC := strings.ReplaceAll(device.Mac, ":", "")
		if len(cutMAC) < 6 {
			return fmt.Errorf("некорректный MAC-адрес: %s", device.Mac)
		}
		cutMAC = strings.ToUpper(cutMAC[len(cutMAC)-6:])
		device.CommonName = fmt.Sprintf("BT-%s-%s", cutMAC, shopper.NumDote)
		check = true
	}

	if shopper.Name == "ovision" {
		device.Name = device.Mac
		check = true
	}
	
	if !check {
		return(fmt.Errorf("Такого клиента не существует: %s", shopper.Name))
	}

	return nil
}


//Загрузить фото для standby режима
func uploadStandbyAsset(ctx context.Context, client *http.Client, baseURL string, accessToken string, filePath string) ([]byte, error) {
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


	url := fmt.Sprintf("%s%s", baseURL, "/pipelineomini/installassets/standby")
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

