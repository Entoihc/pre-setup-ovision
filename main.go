package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Consumer struct {
	Name    string
	NumDote string
}

type Device struct {
	Mac        string `json:"mac_eth"`
	Serial     string
	CommonName string
	Name       string
}

type Url struct {
	protocol     string
	host         string
	portPanel    string
	portPipeline string
}

type securityStatus struct {
	Status struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
	Data struct {
		Hsc              bool   `json:"hsc"`
		Cipf             string `json:"cipf"`
		OpenSSL          string `json:"openssl"`
		OpenVPN_gost     string `json:"openvpn_gost"`
		CryptoTunnel     string `json:"cryptotunnel"`
		LicenseActivated bool   `json:"license_activated"`
	} `json:"data"`
}

func main() {

	// Конфиг
	baseURL := Url{
		protocol:     "http",
		host:         "192.168.88.127",
		portPanel:    "4011",
		portPipeline: "7777",
	}

	pathStandBy := "icon-waiting.png"
	pathWait := "icon-waiting.png"
	pathOpenSSL := "./CIPTonline/openssl-r_1.1.1o-6.10.around_armhf.deb"
	pathOpenVPN := "./CIPTonline/openvpn-gost_2.4.11-5.12_armhf.deb"
	pathStunnel := "./CIPTonline/stunnel-gost_5.60-5.9_armhf.deb"
	pathGmkseed := "./CIPTonline/gmkseed_4.0.0-4.2_armhf.deb"
	licenseCipt := "255D-EJW4-CD58-C36N"

	var curretDevice Device

	shopper := Consumer{
		Name:    "t2",
		NumDote: "120987",
	}

	client := &http.Client{
		Timeout: 30 * time.Minute,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	creds := LoginRequest{
		Username: "cryptouser",
		Password: "Test123@",
	}

	//	Основной цикл

	err := checkPassword(ctx, client, baseURL, &creds)
	if err != nil {
		fmt.Printf("Ошибка проверки пароля: %v\n", err)
		return
	}

	tokenAuth, err := login(ctx, client, baseURL, creds.Username, creds.Password)
	if err != nil {
		fmt.Printf("Ошибка авторизации: %v\n", err)
		return
	}

	//	fmt.Println(tokenAuth.AccessToken)
	//	fmt.Println(tokenAuth.RefreshToken)

	err = getInfoDevice(ctx, client, baseURL, tokenAuth.AccessToken, &curretDevice)
	if err != nil {
		fmt.Printf("fail get info device: %s", err)
		return
	}

	err = getDeviceName(shopper, &curretDevice)
	if err != nil {
		fmt.Printf("fail get info device: %s", err)
		return
	}

	//	fmt.Printf("Name is: %s\n", curretDevice.CommonName)

	payloadRemoteTransaction := RemoteTransactionRequest{
		Enabled:              true,
		DeviceName:           curretDevice.Name,
		DeviceNameIsHostName: false,
		PingURL:              "",
		TimePing:             3,
	}

	another := false
	if another {
		_, err = setRemoteTransactionParameters(ctx, client, baseURL, tokenAuth.AccessToken, payloadRemoteTransaction)
		if err != nil {
			fmt.Printf("Ошибка перевода в режим внешнего управления: %v\n", err)
			return
		}

		_, err = uploadStandbyAsset(ctx, client, baseURL, tokenAuth.AccessToken, pathStandBy)
		if err != nil {
			fmt.Printf("Ошибка загрузки изображения standBy: %v\n", err)
			return
		}

		_, err = uploadWaitAsset(ctx, client, baseURL, tokenAuth.AccessToken, pathWait)
		if err != nil {
			fmt.Printf("Ошибка загрузки изображения для режима ожидания: %v\n", err)
			return
		}

		err = refreshTokens(ctx, client, baseURL, tokenAuth)
		if err != nil {
			fmt.Printf("Ошибка обновления токена: %v\n", err)
			return
		}

		payloadDisplayParameters := DisplayParameters{
			MinDisplayBacklight: 170,
			MaxDisplayBacklight: 210,
			FontSize:            40,
			TextPositionX:       240,
			TextPositionY:       120,
			DebugMode:           true,
		}

		_, err = setDisplayParameters(ctx, client, baseURL, tokenAuth.AccessToken, payloadDisplayParameters)
		if err != nil {
			fmt.Printf("Ошибка изменения параметров дисплея: %v\n", err)
			return
		}

	}

	_, err = uploadCipt(ctx, client, baseURL, tokenAuth.AccessToken, pathOpenSSL, "openssl")
	if err != nil {
		fmt.Printf("Ошибка установки СКЗИ: %v\n", err)
		return
	}

	_, err = uploadCipt(ctx, client, baseURL, tokenAuth.AccessToken, pathOpenVPN, "openvpn")
	if err != nil {
		fmt.Printf("Ошибка установки OpenVPN: %v\n", err)
		return
	}

	_, err = uploadCipt(ctx, client, baseURL, tokenAuth.AccessToken, pathStunnel, "stunnel")
	if err != nil {
		fmt.Printf("Ошибка установки Stunnel: %v\n", err)
		return
	}

	_, err = uploadCipt(ctx, client, baseURL, tokenAuth.AccessToken, pathGmkseed, "gmkseed")
	if err != nil {
		fmt.Printf("Ошибка установки Gmkseed: %v\n", err)
		return
	}

	err = setInitSeed(ctx, client, baseURL)
	if err != nil {
		fmt.Printf("Ошибка инициализации случайного числа: %v\n", err)
		return
	}

	err = activateLicenseOnline(ctx, client, baseURL, licenseCipt)
	if err != nil {
		fmt.Printf("Ошибка активации лицензии: %v\n", err)
		return
	}

}
