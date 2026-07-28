package main

// Скрипт pre-setup-ovision предназначен для предварительной массовой настройки терминалов ovision перед его использованием
// Пример конфигурации для запуска скрипта pre-setup-ovision

//
// GENERAL TYPE AND CONSTS
//

// Информация о клиенте
type Consumer struct {
	Name    string `json:"name"`
	OrgName string `json:"orgName"`
	INN     string `json:"inn"`
}

// Информация о девайсе
type Device struct {
	Mac        string `json:"mac_eth"`
	Name       string
	SerialMac  string
	Serial     string
	CommonName string
	NumDote    string
}

// Ссылка для отправки запроса
type Url struct {
	protocol     string
	host         string
	portPanel    string
	portPipeline string
}

// Пути до утилит необходимых для работы скрипта
type pathUtil struct {
	OpenSSL string
}

// Файл для логов
type Logger struct {
	filename string
}

// Структура для парсинга CSV файла с девайсами
type CSVData struct {
	Header []string
	Rows   []map[string]string
}

//
// PANEL
//
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Текущий статус по устройству
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

// Настройки режима внешнего управления
type RemoteTransactionRequest struct {
	Enabled              bool   `json:"enabled"`
	DeviceName           string `json:"deviceName"`
	DeviceNameIsHostName bool   `json:"deviceNameIsHostName"`
	PingURL              string `json:"pingUrl"`
	TimePing             int    `json:"timePing"`
}

// Настройки дисплея
type DisplayParameters struct {
	MinDisplayBacklight int  `json:"minDisplayBacklight"`
	MaxDisplayBacklight int  `json:"maxDisplayBacklight"`
	FontSize            int  `json:"fontSize"`
	TextPositionX       int  `json:"textPositionX"`
	TextPositionY       int  `json:"textPositionY"`
	DebugMode           bool `json:"debugMode"`
}

//
// CIPT
//
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

type StunnelParametrs struct {
	Stage   string `json:"stage"`
	TlsMode string `json:"tls_mode"`
}
