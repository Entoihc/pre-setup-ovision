package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)



func main() {

	// Конфиг
	IP := "192.168.91.43"
//	portPipeline := "7777"
	portPanel := "4011"
	pathStandBy := "icon-waiting.png"
//	pathWait := "wait.png"
	baseURL := fmt.Sprintf("http://%s:%s", IP, portPanel)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	creds := LoginRequest {
		Username: "cryptouser",
		Password: "Test123@",
	}

	tokenAuth, err := login(ctx, client, baseURL, creds.Username, creds.Password)
	if err != nil {
		fmt.Printf("Ошибка авторизации: %v\n", err)
		return
	}

	// Не рекомендуется печатать токены полностью в реальном приложении.
	fmt.Printf("Access token: %s...\n", tokenAuth.AccessToken)
	fmt.Printf("Refresh token: %s...\n", tokenAuth.RefreshToken)


    _, err = uploadStandbyAsset (ctx, client, baseURL, tokenAuth.AccessToken, pathStandBy)
	if err != nil {
		fmt.Printf("Ошибка загрузки изображения standBy: %v\n", err)
		return
	}

	err = refreshTokens(ctx, client, baseURL, tokenAuth)
	if err != nil {
		fmt.Printf("Ошибка обновления токена: %v\n", err)
		return
	}
}


