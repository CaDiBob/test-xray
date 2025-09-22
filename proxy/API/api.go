package API

import (
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/uuid"
)

func EnsureUserByUUID(id uuid.UUID) error {
	pid := protocol.NewID(id)
	uid := pid.String()

	// троттлинг для конкретного uuid
	rateLimitPerID(uid)

	// Все параллельные вызовы для одного uid схлопываем в один
	_, err, _ := ensureSF.Do(uid, func() (interface{}, error) {
		return nil, ensureUserByUUIDInternal(uid)
	})
	return err
}

func ensureUserByUUIDInternal(uid string) error {
	APIInfof("ensure: received UUID %s", uid)

	// Биллинг
	user, err := GetBillingUser(uid)
	if err != nil {
		return APIWrapErr("ensure: billing lookup failed", err)
	}
	APIInfof("ensure: billing user fetched for %s", uid)

	payload := buildPayloadFromBilling(user)

	username := user.Username
	if username == "" {
		if v, ok := payload["username"].(string); ok && v != "" {
			username = v
		}
	}

	// Проверяем, есть ли такой пользователь в Marzban
	exists, err := marzbanUserExists(username)
	if err != nil {
		return APIWrapErr("ensure: failed to check user in Marzban", err)
	}

	// Создаём или обновляем пользователя в Marzban
	if exists {
		APIInfof("ensure: user exists in Marzban, updating...")
		if err := updateUserInMarzban(username, user); err != nil {
			return APIWrapErr("ensure: failed to update user in Marzban", err)
		}
	} else {
		APIInfof("ensure: user not found in Marzban, creating...")

		if err := createUserInMarzban(payload); err != nil {
			// На случай гонки между exists и create в другой горутине:
			es := err.Error()
			
			if es == "marzban create user failed: {\"detail\":\"User already exists\"}" ||
				es == "marzban create user failed: {\"detail\":\"User Already Exists\"}" {
				APIInfof("ensure: got 409 on create, switching to update...")
				if err2 := updateUserInMarzban(username, user); err2 != nil {
					return APIWrapErr("ensure: update after 409 failed", err2)
				}
			} else {
				return APIWrapErr("ensure: failed to create user in Marzban", err)
			}
		}
	}

	APIInfof("ensure: user %s synced to Marzban", uid)
	return nil
}