package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/tidwall/gjson"
)

var logger *Logger
var refresh time.Time

func main() {

	// Версия ПО, под которое разработан скрипт - 1.11.6

	//
	// Чтение конфига
	// Решил конфиг не разбивать на структуры, больно много переписывать впустую, пока буду работать напрямую с json через gjson
	//

	data, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Println("Ошибка чтения конфигурационного файла:", err)
		return
	}

	// Преобразуем в строку и отдаем gjson
	configJSON := string(data)
	pathLogFile := gjson.Get(configJSON, "path.pathLogFile").String()
	if pathLogFile == "" {
		fmt.Printf("%s - Название лог-файла не может быть пустым: %s", time.Now().Format("2006-01-02 15:04:05"), "")
		return
	}

	// Выводим информацию куда будут записаны логи
	fmt.Println("Все логи будут записаны в файл:", pathLogFile)
	logger := NewLogger(pathLogFile, true)

	// Создаем переменные для вывода инфомрации об устройствах
	pathOutput := fmt.Sprintf("./out/Info-device-%s", time.Now().Format("2006-01-02 15:04"))
	output := NewLogger(pathOutput, false)
	output.Log(fmt.Sprintf("%s,%s,%s,%s,%s,%s", "IP", "MAC", "Name", "NumDote", "CommonName", "Serial"))

	err = logger.Log("НАЧАЛО РАБОТЫ СКРИПТА ==================================================")
	if err != nil {
		fmt.Printf("%s - Не удалось записать лог в файл: %s", time.Now().Format("2006-01-02 15:04:05"), "")
	}

	shopper := Consumer{
		Name:    gjson.Get(configJSON, "consumer.Name").String(),
		OrgName: gjson.Get(configJSON, "consumer.OrgName").String(),
		INN:     gjson.Get(configJSON, "consumer.INN").String(),
	}

	shopperString, err := json.Marshal(shopper)
	err = logger.Log(string(shopperString))
	if err != nil {
		fmt.Printf("%s - Не удалось записать лог в файл: %s", time.Now().Format("2006-01-02 15:04:05"), "")
	}

	creds := LoginRequest{
		Username: gjson.Get(configJSON, "login.Username").String(),
		Password: gjson.Get(configJSON, "login.Password").String(),
	}

	client := &http.Client{
		Timeout: 30 * time.Minute,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	pathListDevices := gjson.Get(configJSON, "path.pathListDevices").String()
	listDevice, err := ParseCSVToMaps(pathListDevices)
	if err != nil {
		fmt.Printf("Ошибка: %v\n", err)
		return
	}

	var device Device
	baseURL := Url{
		protocol:     "http",
		portPanel:    "4011",
		portPipeline: "7777",
	}

	//
	// Вывод того что будет делать скрипт и ожидание подтверждения пользователя
	//
	fmt.Println("Будут выполнены следующие действия:")
	readInstalledServiceAction := gjson.Get(configJSON, "allAction.readInstalledService.action").Bool()
	if readInstalledServiceAction {
		fmt.Println("readInstalledService")
	}

	setRemoteTransactionParametersAction := gjson.Get(configJSON, "allAction.setRemoteTransactionParameters.action").Bool()
	if setRemoteTransactionParametersAction {
		fmt.Println("setRemoteTransactionParameters")
	}

	uploadPhotoAction := gjson.Get(configJSON, "allAction.uploadPhoto.action").Bool()
	if uploadPhotoAction {
		fmt.Println("uploadPhoto")
	}

	setDisplayParametersAction := gjson.Get(configJSON, "allAction.setDisplayParameters.action").Bool()
	if setDisplayParametersAction {
		fmt.Println("setDisplayParameters")
	}

	setCIPTAction := gjson.Get(configJSON, "allAction.setCIPT.action").Bool()
	if setCIPTAction {
		fmt.Println("setCIPT")
	}

	initSeedAction := gjson.Get(configJSON, "allAction.initSeed.action").Bool()
	if initSeedAction {
		fmt.Println("initSeed")
	}

	downloadInfoFileAction := gjson.Get(configJSON, "allAction.downloadInfoFile.action").Bool()
	if downloadInfoFileAction {
		fmt.Println("downloadInfoFile")
	}

	activateLicenseAction := gjson.Get(configJSON, "allAction.activateLicense.action").Bool()
	if activateLicenseAction {
		fmt.Println("uploadLicenseOffline")
	}

	issueRequestCsrAction := gjson.Get(configJSON, "allAction.issueRequestCsr.action").Bool()
	if issueRequestCsrAction {
		fmt.Println("issueRequestCsr")
	}

	uploadCertAction := gjson.Get(configJSON, "allAction.uploadCert.action").Bool()
	if uploadCertAction {
		fmt.Println("uploadCert")
	}

	setOpenVPNParametersAction := gjson.Get(configJSON, "allAction.setOpenVPNParameters.action").Bool()
	if setOpenVPNParametersAction {
		fmt.Println("setOpenVPNParameters")
	}

	setStunnelParametersAction := gjson.Get(configJSON, "allAction.setStunnelParameters.action").Bool()
	if setStunnelParametersAction {
		fmt.Println("setStunnelParameters")
	}

	poweroffAction := gjson.Get(configJSON, "allAction.poweroff.action").Bool()
	if poweroffAction {
		fmt.Println("poweroff")
	}

	//
	// По каждому устройству выполняются действия указанные в конфиге
	//
	for i, row := range listDevice.Rows {
		baseURL.host = row["IP"]
		device.NumDote = row["NumDote"]
		logger.Log("=========")
		logger.Log(fmt.Sprintf("Девайс %d - IP:%s, NumDote:%s", i+1, baseURL.host, device.NumDote))

		err := checkPassword(ctx, client, baseURL, &creds)
		if err != nil {
			logger.Log(fmt.Sprintf("Ошибка проверки пароля - %s", err))
			// continue
		}

		tokenAuth, err := login(ctx, client, baseURL, creds.Username, creds.Password)
		if err != nil {
			logger.Log(fmt.Sprintf("Ошибка авторизации - %s", err))
			// continue
		} else {
			err = refreshTokens(ctx, client, baseURL, tokenAuth)
			if err != nil {
				logger.Log(fmt.Sprintf("Ошибка обновления токена - %s", err))
			}

			err = getInfoDevice(ctx, client, baseURL, tokenAuth.AccessToken, &device)
			if err != nil {
				logger.Log(fmt.Sprintf("Не удалось получить mac-адрес девайса - %s", err))
			}

			err = getDeviceName(&shopper, &device)
			if err != nil {
				logger.Log(fmt.Sprintf("Не удалось получить записать информацию девайса - %s", err))
			}

			logger.Log(fmt.Sprintf("MAC:%s, Name:%s, CN:%s", device.Mac, device.Name, device.CommonName))

			err = refreshTokens(ctx, client, baseURL, tokenAuth)
			if err != nil {
				logger.Log(fmt.Sprintf("Ошибка обновления токена - %s", err))
			}
		}

		if readInstalledServiceAction {
			logger.Log("Получение списка установленных пакетов")

			inst, err := getIstalledPackage(ctx, client, baseURL, tokenAuth.AccessToken)
			if err != nil {
				logger.Log(fmt.Sprintf("Ошибка получения списка установленных пакетов - %s", err))
				continue
			}
			logger.Log(fmt.Sprintf("Установленные пакеты:%s", inst))

		}

		if setRemoteTransactionParametersAction {
			logger.Log("Изменение параметров внешнего распознавания")
			payloadRemoteTransaction := RemoteTransactionRequest{
				Enabled:              gjson.Get(configJSON, "allAction.setRemoteTransactionParameters.RemoteTransactionParameters.Enabled").Bool(),
				DeviceName:           device.Name,
				DeviceNameIsHostName: gjson.Get(configJSON, "allAction.setRemoteTransactionParameters.RemoteTransactionParameters.DeviceNameIsHostName").Bool(),
				PingURL:              gjson.Get(configJSON, "allAction.setRemoteTransactionParameters.RemoteTransactionParameters.PingURL").String(),
				TimePing:             int(gjson.Get(configJSON, "allAction.setRemoteTransactionParameters.RemoteTransactionParameters.DeviceNameIsHostName").Int()),
			}

			_, err = setRemoteTransactionParameters(ctx, client, baseURL, tokenAuth.AccessToken, payloadRemoteTransaction)
			if err != nil {
				logger.Log(fmt.Sprintf("Ошибка изменения настроек режима внешнего управления - %s", err))
			}

		}

		if uploadPhotoAction {
			logger.Log("Загрузка новых ассетов")
			pathStandBy := gjson.Get(configJSON, "allAction.uploadPhoto.pathStandBy").String()
			pathWait := gjson.Get(configJSON, "allAction.uploadPhoto.pathWait").String()

			_, err = uploadStandbyAsset(ctx, client, baseURL, tokenAuth.AccessToken, pathStandBy)
			if err != nil {
				logger.Log(fmt.Sprintf("Ошибка загрузки изображения ожидания начала транзакции: %s", err))
			}

			_, err = uploadWaitAsset(ctx, client, baseURL, tokenAuth.AccessToken, pathWait)
			if err != nil {
				logger.Log(fmt.Sprintf("Ошибка загрузки изображения для режима ожидания окончания транзакции: %s", err))
			}
		}

		if setDisplayParametersAction {
			logger.Log("Изменение параметров дисплея")
			payloadDisplayParameters := DisplayParameters{
				MinDisplayBacklight: int(gjson.Get(configJSON, "allAction.setDisplayParameters.display.MinDisplayBacklight").Int()),
				MaxDisplayBacklight: int(gjson.Get(configJSON, "allAction.setDisplayParameters.display.MaxDisplayBacklight").Int()),
				FontSize:            int(gjson.Get(configJSON, "allAction.setDisplayParameters.display.FontSize").Int()),
				TextPositionX:       int(gjson.Get(configJSON, "allAction.setDisplayParameters.display.TextPositionX").Int()),
				TextPositionY:       int(gjson.Get(configJSON, "allAction.setDisplayParameters.display.TextPositionY").Int()),
				DebugMode:           gjson.Get(configJSON, "allAction.setDisplayParameters.display.DebugMode").Bool(),
			}

			_, err = setDisplayParameters(ctx, client, baseURL, tokenAuth.AccessToken, payloadDisplayParameters)
			if err != nil {
				logger.Log(fmt.Sprintf("Ошибка изменения настроек дисплея: %s", err))
			}
		}

		err = refreshTokens(ctx, client, baseURL, tokenAuth)
		if err != nil {
			logger.Log(fmt.Sprintf("Ошибка обновления токена - %s", err))
		}

		if setCIPTAction {
			logger.Log("Установка СКЗИ")
			online := gjson.Get(configJSON, "allAction.setCIPT.online").Bool()

			var pathOpenSSL, pathOpenVPN, pathStunnel, pathGmkseed string

			if online {
				pathOpenSSL = gjson.Get(configJSON, "allAction.setCIPT.pathOpenSSLOnline").String()
				pathOpenVPN = gjson.Get(configJSON, "allAction.setCIPT.pathOpenVPNOnline").String()
				pathStunnel = gjson.Get(configJSON, "allAction.setCIPT.pathStunnelOnline").String()
				pathGmkseed = gjson.Get(configJSON, "allAction.setCIPT.pathGmkseedOnline").String()

			} else {
				pathOpenSSL = gjson.Get(configJSON, "allAction.setCIPT.pathOpenSSLOffline").String()
				pathOpenVPN = gjson.Get(configJSON, "allAction.setCIPT.pathOpenVPNOffline").String()
				pathStunnel = gjson.Get(configJSON, "allAction.setCIPT.pathStunnelOffline").String()
				pathGmkseed = gjson.Get(configJSON, "allAction.setCIPT.pathGmkseedOffline").String()
			}

			_, err = uploadCipt(ctx, client, baseURL, tokenAuth.AccessToken, pathOpenSSL, "openssl")
			if err != nil {
				logger.Log(fmt.Sprintf("Ошибка установки OpenSSL: %s", err))
			}

			_, err = uploadCipt(ctx, client, baseURL, tokenAuth.AccessToken, pathOpenVPN, "openvpn")
			if err != nil {
				logger.Log(fmt.Sprintf("Ошибка установки OpenVPN: %s", err))
			}

			_, err = uploadCipt(ctx, client, baseURL, tokenAuth.AccessToken, pathStunnel, "stunnel")
			if err != nil {
				logger.Log(fmt.Sprintf("Ошибка установки Stunnel: %s", err))
			}

			_, err = uploadCipt(ctx, client, baseURL, tokenAuth.AccessToken, pathGmkseed, "gmkseed")
			if err != nil {
				logger.Log(fmt.Sprintf("Ошибка установки Gmkseed: %s", err))
			}
		}

		if activateLicenseAction {
			logger.Log("Активация СКЗИ")
			online := gjson.Get(configJSON, "allAction.activateLicense.online").Bool()

			if online {
				logger.Log("Активация онлайн лицензии СКЗИ")

				licenseCipt := gjson.Get(configJSON, "allAction.activateLicense.license").String()
				err = activateLicenseOnline(ctx, client, baseURL, tokenAuth.AccessToken, licenseCipt)
				if err != nil {
					logger.Log(fmt.Sprintf("Ошибка активации лицензии: %s", err))
				}
			}

			if !online {
				logger.Log("Активация оффлайн лицензии СКЗИ пока не готова")
			}
		}

		if initSeedAction {
			logger.Log("Генерация случайного числа")
			err = setInitSeed(ctx, client, baseURL)
			if err != nil {
				logger.Log(fmt.Sprintf("Ошибка генерации случайного числа: %s", err))
			}
		}

		err = refreshTokens(ctx, client, baseURL, tokenAuth)
		if err != nil {
			logger.Log(fmt.Sprintf("Ошибка обновления токена - %s", err))
		}

		if issueRequestCsrAction {
			logger.Log("Скачивание запроса на сертификат")
			pathDirCert := gjson.Get(configJSON, "allAction.issueRequestCsr.pathDirCert").String()
			_, err = issueRequestCertificate(ctx, client, baseURL, tokenAuth.AccessToken, &device, &shopper, pathDirCert)
			if err != nil {
				logger.Log(fmt.Sprintf("Ошибка скачивания запроса на сертификат: %s", err))
			}
		}

		// for {
		// 	if uploadCertAction {
		// 		logger.Log("Загрузка сертификата для OpenVPN")
		// 		pathDirCert := gjson.Get(configJSON, "allAction.uploadCert.pathDirCert").String()

		// 		certs, err := readCertsCommonName(pathDirCert, gjson.Get(configJSON, "path.pathOpenSSL").String())
		// 		if err != nil {
		// 			logger.Log(fmt.Sprintf("Ошибка чтения папки с сертификатами - %s", err))
		// 			break
		// 		}

		// 		cert, err := findCertByCommonName(certs, device.CommonName)
		// 		if err != nil {
		// 			logger.Log(fmt.Sprintf("Ошибка поиска подходящего сертификата - %s", err))
		// 			break
		// 		}

		// 		_, err = uploadCaCertificate(ctx, client, baseURL, tokenAuth.AccessToken, cert)
		// 		if err != nil {
		// 			logger.Log(fmt.Sprintf("Ошибка загрузки сертификата: %s", err))
		// 			break
		// 		}
		// 	}
		// 	break
		// }

		// if setOpenVPNParametersAction {
		// 	logger.Log("Установка параметров OpenVPN")

		// 	ip := gjson.Get(json, "openvpn.OpenVpnAddress.0.ip").String()
		// 	port := gjson.Get(json, "openvpn.OpenVpnAddress.0.port").Int()
		// 	protocol := gjson.Get(json, "openvpn.Protocol").String()
		// 	tunMTU := gjson.Get(json, "openvpn.TunMTU").String()
		// 	crlVerify := gjson.Get(json, "openvpn.CrlVerify").Bool()

		// 	OpenVpnAddress := OpenVpnAddress{
		// 		IP:  gjson.Get(configJSON, "allAction.setOpenVPNParameters.openvpn.OpenVpnAddress.ip[]").String(),
		// 	}

		// 	OpenVpnParametrs := OpenVpnParametrs{
		// 		Addresses:=
		// 	}

		// 	_, err = setOpenVpnParametrs(ctx, client, baseURL, tokenAuth.AccessToken, OpenVpnParametrs)
		// }

		if poweroffAction {
			logger.Log("Выключение устройства")
			err = PowerOff(baseURL)
			if err != nil {
				logger.Log("Не удалось выключить устройство")
			}
		}
		output.Log(fmt.Sprintf("%s,%s,%s,%s,%s,%s", baseURL.host, device.Mac, device.Name, device.NumDote, device.CommonName, device.Serial))
	}

	return
}
