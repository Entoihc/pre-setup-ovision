package main

// Скрипт pre-setup-ovision предназначен для предварительной массовой настройки терминалов ovision перед его использованием
// Пример конфигурации для запуска скрипта pre-setup-ovision

//
// GENERAL TYPE AND CONSTS
//

// Action
type Action struct {
	SetRemoteTransactionParameters bool `json:"allAction.setRemoteTransactionParameters.action"`
	UploadPhoto                    bool
	SetDisplayParameters           bool
	SetCIPT                        bool
	DownloadInfoFile               bool
	UploadLicenseOffline           bool
	IssueRequestCsr                bool
	UploadCert                     bool
	SetOpenVPNParameters           bool
	SetStunnelParameters           bool
}

// Информация о клиенте
type Consumer struct {
	Name    string
	OrgName string
	INN     string
}

type Device struct {
	Mac        string `json:"mac_eth"`
	Name       string
	Serial     string
	CommonName string
	NumDote    string
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

type pathUtil struct {
	OpenSSL string
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

type RemoteTransactionRequest struct {
	Enabled              bool   `json:"enabled"`
	DeviceName           string `json:"deviceName"`
	DeviceNameIsHostName bool   `json:"deviceNameIsHostName"`
	PingURL              string `json:"pingUrl"`
	TimePing             int    `json:"timePing"`
}

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
