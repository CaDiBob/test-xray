package API

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}

	muToken         sync.Mutex
	cachedToken     string
	tokenExpiryTime time.Time

	marzbanBaseURL  = getenv("MARZBAN_BASE_URL", "http://45.144.49.228:8000")
	marzbanUsername = getenv("MARZBAN_USERNAME", "XRAY")
	marzbanPassword = getenv("MARZBAN_PASSWORD", "XRAY-SECRET-PASSWORD")
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getAccessToken() (string, error) {
	muToken.Lock()
	defer muToken.Unlock()

	if cachedToken != "" && time.Now().Before(tokenExpiryTime) {
		return cachedToken, nil
	}

	APIInfof("auth: requesting admin token from Marzban ...")

	resp, err := httpClient.PostForm(marzbanBaseURL+"/api/admin/token", url.Values{
		"grant_type":    {"password"},
		"username":      {marzbanUsername},
		"password":      {marzbanPassword},
		"client_id":     {"string"},
		"client_secret": {"string"},
	})
	if err != nil {
		APIErrorf("auth request failed: %v", err)
		return "", fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		APIErrorf("auth failed: status=%d body=%s", resp.StatusCode, string(b))
		return "", fmt.Errorf("auth failed: %s", string(b))
	}

	var auth struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		APIErrorf("auth decode failed: %v", err)
		return "", fmt.Errorf("auth decode failed: %w", err)
	}

	cachedToken = auth.AccessToken

	return cachedToken, nil
}

// doAuthedJSON — отправить JSON-запрос с Bearer токеном.
// При 401 — единоразово обновляет токен и повторяет запрос.
func doAuthedJSON(method, path string, body []byte) (*http.Response, error) {
	token, err := getAccessToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, marzbanBaseURL+path, bytes.NewBuffer(body))
	if err != nil {
		APIErrorf("request create failed: %v", err)
		return nil, fmt.Errorf("request create failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		APIErrorf("request failed: %v", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		APIInfof("auth: 401 Unauthorized, refreshing token and retrying")
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		muToken.Lock()
		cachedToken = ""
		tokenExpiryTime = time.Time{}
		muToken.Unlock()

		token, err = getAccessToken()
		if err != nil {
			APIErrorf("reauth failed: %v", err)
			return nil, fmt.Errorf("reauth failed: %w", err)
		}

		req2, err := http.NewRequest(method, marzbanBaseURL+path, bytes.NewBuffer(body))
		if err != nil {
			APIErrorf("request2 create failed: %v", err)
			return nil, fmt.Errorf("request2 create failed: %w", err)
		}
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "Bearer "+token)

		return httpClient.Do(req2)
	}

	return resp, nil
}
