package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)


type Consumer struct {
	Name string
	NumDote string 

}

type Device struct{
	Mac string `json:"mac_eth"`
	Serial string
	CommonName string
	Name string
}


func main() {



	// Конфиг
	IP := "192.168.91.43"
//	portPipeline := "7777"
	portPanel := "4011"
	pathStandBy := "icon-waiting.png"
//	pathWait := "wait.png"
	baseURL := fmt.Sprintf("http://%s:%s", IP, portPanel)

	var curretDevice Device

	shopper := Consumer {
		Name: "t2",
		NumDote: "120987",
	}

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

	err = getInfoDevice(ctx, client, baseURL, tokenAuth.AccessToken, &curretDevice)
	if err != nil {
		fmt.Printf("fail get info device: %w", err)
		return
	}

	err = getDeviceName(shopper, &curretDevice)
	if err != nil {
		fmt.Printf("fail get info device: %w", err)
		return
	}

	fmt.Printf("Name is: %s\n", curretDevice.CommonName)


	payload := RemoteTransactionRequest {
		Enabled: true,
		DeviceName: curretDevice.Name,
		DeviceNameIsHostName: false,
		PingURL: "",
		TimePing: 3,
	}

	_, err = sendRemoteTransaction(ctx, client, baseURL, tokenAuth.AccessToken, payload)
	if err != nil {
		fmt.Printf("Ошибка перевода в режим внешнего управления: %v\n", err)
		return
	}

	_, err = uploadStandbyAsset(ctx, client, baseURL, tokenAuth.AccessToken, pathStandBy)
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


