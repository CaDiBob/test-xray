package API

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func buildPayloadFromBilling(bu BillingUser) map[string]any {
	username := bu.Username
	if username == "" {
		username = "manual"
	}
	status := bu.Status
	if status == "" {
		status = "active"
	}

	vlessID := bu.UUID
	if bu.Proxies.VLESS.ID != "" {
		vlessID = bu.Proxies.VLESS.ID
	}
	flow := bu.Proxies.VLESS.Flow

	var expire any = nil
	if bu.Expire != nil && *bu.Expire != "" {
		s := *bu.Expire
		var (
			t   time.Time
			err error
		)
		layouts := []string{time.RFC3339Nano, time.RFC3339}
		for _, layout := range layouts {
			t, err = time.Parse(layout, s)
			if err == nil {
				break
			}
		}
		if err == nil {
			expire = t.Unix()
		} else {
			expire = nil
		}
	}

	payload := map[string]any{
		"proxies": map[string]any{
			"vless": map[string]string{
				"id":   vlessID,
				"flow": flow,
			},
		},
		"expire":                    expire,
		"data_limit":                bu.DataLimit,
		"data_limit_reset_strategy": nonEmptyOrDefault(bu.DataLimitResetStrategy, "no_reset"),
		"inbounds": map[string]any{
			"vless": []string{"VLESS TCP REALITY"},
		},
		"note":                    bu.Note,
		"sub_updated_at":          nil,
		"sub_last_user_agent":     nil,
		"online_at":               nil,
		"on_hold_expire_duration": bu.OnHoldExpireDuration,
		"on_hold_timeout":         bu.OnHoldTimeout,
		"auto_delete_in_days":     bu.AutoDeleteInDays,
		"next_plan":               nil,
		"username":                username,
		"status":                  status,
		"used_traffic":            0,
		"lifetime_used_traffic":   0,
		"created_at":              time.Now().Format(time.RFC3339),

		"links":             []string{},
		"subscription_url":  "",
		"excluded_inbounds": map[string]any{"vless": []string{}, "shadowsocks": []string{}},
		"admin":             nil,
	}
	return payload
}

func nonEmptyOrDefault(s, def string) string {
	if s != "" {
		return s
	}
	return def
}

func marzbanUserExists(username string) (bool, error) {
	rel := "/api/user/" + url.PathEscape(username)
	resp, err := doAuthedJSON("GET", rel, nil)
	if err != nil {
		return false, APIWrapErr("marzbanUserExists request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		APIDebugf("marzbanUserExists: %s not found", username)
		return false, nil
	}
	if resp.StatusCode == http.StatusOK {
		APIDebugf("marzbanUserExists: %s exists", username)
		return true, nil
	}

	body, _ := io.ReadAll(resp.Body)
	APIErrorf("marzbanUserExists unexpected status=%d body=%s", resp.StatusCode, string(body))
	return false, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
}

func createUserInMarzban(payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return APIWrapErr("marshal payload failed", err)
	}

	resp, err := doAuthedJSON("POST", "/api/user", body)
	if err != nil {
		return APIWrapErr("marzban create user request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		APIInfof("marzban create user failed: status=%d body=%s", resp.StatusCode, string(b))
		return fmt.Errorf("marzban create user failed: %s", string(b))
	}

	APIInfof("marzban: user created/updated successfully")
	return nil
}

func buildUpdatePayloadFromBilling(bu BillingUser) map[string]any {
	var expire any = nil
	if bu.Expire != nil && *bu.Expire != "" {
		s := *bu.Expire
		var (
			t   time.Time
			err error
		)
		layouts := []string{time.RFC3339Nano, time.RFC3339}
		for _, layout := range layouts {
			t, err = time.Parse(layout, s)
			if err == nil {
				break
			}
		}
		if err == nil {
			expire = t.Unix()
		}
	}

	vlessID := bu.UUID
	if bu.Proxies.VLESS.ID != "" {
		vlessID = bu.Proxies.VLESS.ID
	}
	flow := bu.Proxies.VLESS.Flow

	inbounds := map[string]any{
		"vless": []string{"VLESS TCP REALITY"},
	}

	update := map[string]any{
		"data_limit":                bu.DataLimit,
		"data_limit_reset_strategy": nonEmptyOrDefault(bu.DataLimitResetStrategy, "no_reset"),
		"expire":                    expire,
		"inbounds":                  inbounds,
		"next_plan": map[string]any{
			"add_remaining_traffic": false,
			"data_limit":            0,
			"expire":                0,
			"fire_on_either":        true,
		},
		"note":                    bu.Note,
		"on_hold_expire_duration": bu.OnHoldExpireDuration,
		"on_hold_timeout":         bu.OnHoldTimeout,
		"proxies": map[string]any{
			"vless": map[string]string{
				"id":   vlessID,
				"flow": flow,
			},
		},
		"status": nonEmptyOrDefault(bu.Status, "active"),
	}
	return update
}

func updateUserInMarzban(username string, bu BillingUser) error {
	bodyMap := buildUpdatePayloadFromBilling(bu)
	body, err := json.Marshal(bodyMap)
	if err != nil {
		return APIWrapErr("marshal update payload failed", err)
	}

	rel := "/api/user/" + url.PathEscape(username)
	resp, err := doAuthedJSON("PUT", rel, body)
	if err != nil {
		return APIWrapErr("marzban update user request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		APIInfof("marzban update user failed: status=%d body=%s", resp.StatusCode, string(b))
		return fmt.Errorf("marzban update user failed: %s", string(b))
	}

	APIInfof("marzban: user updated successfully")
	return nil
}