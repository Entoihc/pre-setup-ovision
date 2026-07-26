package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Загрузить СКЗИ через веб-панель
// Загрузить СКЗИ через веб-панель
func uploadCipt(ctx context.Context, client *http.Client, baseURL Url, accessToken string, filePath string) ([]byte, error) {
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

	return responseBody, nil
}
