package API


type BillingUser struct {
	UUID     string `json:"id"`     // Потом поменяем на VLESS UUID, когда обновим ручку
	Username string `json:"username"`
	Status   string `json:"status"`

	// Профили прокси (Пока ток VLESS)
	Proxies struct {
		VLESS struct {
			ID   string `json:"id"`
			Flow string `json:"flow"`
		} `json:"vless"`
	} `json:"proxies"`

	Expire                    *string `json:"expire,omitempty"`
	DataLimit                 *int64  `json:"data_limit,omitempty"`
	DataLimitResetStrategy    string  `json:"data_limit_reset_strategy,omitempty"`
	Note                      string  `json:"note,omitempty"`
	OnHoldExpireDuration      *int64  `json:"on_hold_expire_duration,omitempty"`
	OnHoldTimeout             *int64  `json:"on_hold_timeout,omitempty"`
	AutoDeleteInDays          *int64  `json:"auto_delete_in_days,omitempty"`
} 