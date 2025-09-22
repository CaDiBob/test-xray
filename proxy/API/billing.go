package API

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	billingBaseURL = getenv("BILLING_BASE_URL", "https://stage.ssvpnapp.win")
	billingToken   = getenv("BILLING_TOKEN", "639ca51448ef68be6a473170e24bb7443e704269")
	billingClient  = &http.Client{Timeout: 8 * time.Second}
)

func GetBillingUser(uuid string) (BillingUser, error) {
	var out BillingUser

	url := fmt.Sprintf("%s/api/v1/marzban-client/%s", billingBaseURL, uuid)
	APIInfof("billing: request user %s", uuid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return out, APIWrapErr("billing request create failed", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Token "+billingToken)

	resp, err := billingClient.Do(req)
	if err != nil {
		return out, APIWrapErr("billing request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		APIInfof("billing: user %s not found", uuid)
		return out, fmt.Errorf("billing: user %s not found", uuid)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		APIInfof("billing error: status=%d body=%s", resp.StatusCode, string(b))
		return out, fmt.Errorf("billing error: %s", string(b))
	}

	APIInfof("billing: user %s exists", uuid)

	var result struct {
		Client struct {
			ID       string `json:"id"`
			EndDate  string `json:"end_date"`
			Username string `json:"username"`
			Status   string `json:"status"`
		} `json:"client"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return out, APIWrapErr("billing decode failed", err)
	}

	if result.Client.Status == "locked" {
		APIInfof("billing: user %s is locked on Billing", uuid)
		return out, fmt.Errorf("Failed: user %s is locked on Billing", uuid)
	}

	out.UUID = result.Client.ID
	out.Username = result.Client.Username
	out.Status = result.Client.Status
	out.Expire = &result.Client.EndDate

	APIInfof("billing: fetched user %s username=%s status=%s", uuid, out.Username, out.Status)
	return out, nil
}