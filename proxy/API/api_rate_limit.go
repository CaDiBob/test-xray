package API

import (
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	perIDMu   sync.Mutex
	lastCall  = make(map[string]time.Time)
	minGap    = time.Duration(getenvInt("API_MIN_GAP_MS", 5000)) * time.Millisecond // 5s по умолчанию
)

func getenvInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

// ждёт, чтобы второй и последующие вызовы для одного id шли не чаще minGap
func rateLimitPerID(id string) {
	var wait time.Duration

	perIDMu.Lock()
	now := time.Now()
	if last, ok := lastCall[id]; ok {
		elapsed := now.Sub(last)
		if elapsed < minGap {
			wait = minGap - elapsed
		}
	}
	if wait == 0 {
		lastCall[id] = now
		perIDMu.Unlock()
		return
	}
	perIDMu.Unlock()

	APIInfof("rate: uid=%s delaying %v", id, wait)
	time.Sleep(wait)

	perIDMu.Lock()
	lastCall[id] = time.Now()
	perIDMu.Unlock()
}