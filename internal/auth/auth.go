// Package auth 处理认证相关逻辑，包括签名计算和登录
package auth

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"SF/internal/api"
)

const (
	SALT        = "lPQDb9AKO7$LjkPG"
	DeviceToken = "910D166A-736E-3231-8B21-8D12DFD75F16"
)

// Manager 管理认证状态
type Manager struct {
	client *api.Client
	nonce  string
}

// NewManager 创建新的认证管理器
func NewManager(client *api.Client) *Manager {
	return &Manager{
		client: client,
	}
}

// InitNonce 初始化 nonce 并验证可用性
func (m *Manager) InitNonce() error {
	for {
		m.nonce = strings.ToUpper(uuid.New().String())
		timestamp := time.Now().UnixMilli()
		sign := m.GetSign(m.nonce, timestamp, DeviceToken)

		m.client.SetHeader("sfsecurity", fmt.Sprintf("nonce=%s&timestamp=%d&devicetoken=%s&sign=%s", m.nonce, timestamp, DeviceToken, sign))
		m.client.SetHeader("user-agent", fmt.Sprintf("boluobao/5.1.54(android;35)/OPPO/%s/OPPO", strings.ToLower(DeviceToken)))

		resp, err := m.client.Request("GET", api.BuildURL("/Chaps/8436696?expand=content%2Cexpand.content"), nil)
		if err != nil {
			continue
		}

		if api.GetHTTPCode(resp) != 417 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// GetSign 计算 API 签名
func (m *Manager) GetSign(nonce string, timestamp int64, deviceToken string) string {
	longNonce := []byte(strings.Repeat(nonce, 4))

	indexCalc := func(x int) int {
		x0 := int(longNonce[x])
		x17 := x0 / 0x24
		return x0 - x17*0x24
	}

	offset1 := indexCalc(1)
	offset2 := indexCalc(2)
	offset3 := indexCalc(3)
	offset4 := indexCalc(4)

	nonceReorder := make([]byte, 0, 101)
	nonceReorder = append(nonceReorder, longNonce[offset1:offset1+13]...)
	nonceReorder = append(nonceReorder, longNonce[offset2:offset2+16]...)
	nonceReorder = append(nonceReorder, longNonce[offset3:offset3+36]...)
	nonceReorder = append(nonceReorder, longNonce[offset4:offset4+36]...)

	authString := []byte(fmt.Sprintf("%d%s%s%s", timestamp, SALT, deviceToken, nonce))

	result := make([]byte, 101)
	for i := 0; i < 101; i++ {
		result[i] = byte((int(authString[i]) + int(nonceReorder[i])) >> 1)
	}

	// reorder: D + A + C + B
	lens := []int{13, 16, 36, 36}
	A := result[0:lens[0]]
	B := result[lens[0] : lens[0]+lens[1]]
	C := result[lens[0]+lens[1] : lens[0]+lens[1]+lens[2]]
	D := result[lens[0]+lens[1]+lens[2]:]

	reordered := make([]byte, 0, 101)
	reordered = append(reordered, D...)
	reordered = append(reordered, A...)
	reordered = append(reordered, C...)
	reordered = append(reordered, B...)

	// symbol_add_19
	final := make([]byte, 101)
	for i := 0; i < 101; i++ {
		charCode := int(reordered[i])

		if charCode < 0x30 {
			newCode := charCode + 19
			if 0x39 < newCode && newCode < 0x41 {
				final[i] = 0x39
			} else {
				final[i] = byte(newCode)
			}
		} else if 0x39 < charCode && charCode < 0x41 {
			final[i] = byte(charCode + 19)
		} else if 0x5A < charCode && charCode < 0x61 {
			final[i] = byte(charCode + 19)
		} else {
			final[i] = reordered[i]
		}
	}

	hash := md5.Sum(final)
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}

// RefreshSecurityHeader 刷新安全请求头
func (m *Manager) RefreshSecurityHeader() {
	timestamp := time.Now().UnixMilli()
	sign := m.GetSign(m.nonce, timestamp, DeviceToken)
	m.client.SetHeader("sfsecurity", fmt.Sprintf("nonce=%s&timestamp=%d&devicetoken=%s&sign=%s", m.nonce, timestamp, DeviceToken, sign))
}

// Login 使用用户名密码登录并返回 cookie
func (m *Manager) Login(username, password string) (string, error) {
	m.RefreshSecurityHeader()

	data := map[string]string{
		"password": password,
		"shuMeiId": "",
		"username": username,
	}
	body, _ := json.Marshal(data)

	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("POST", api.BuildURL("/sessions"), bytes.NewReader(body))

	// 复制请求头
	headers := map[string]string{
		"Host":            "api.sfacg.com",
		"accept-charset":  "UTF-8",
		"authorization":   "Basic YW5kcm9pZHVzZXI6MWEjJDUxLXl0Njk7KkFjdkBxeHE=",
		"accept":          "application/vnd.sfacg.api+json;version=1",
		"accept-encoding": "gzip",
		"Content-Type":    "application/json; charset=UTF-8",
		"sfsecurity":      m.client.GetHeader("sfsecurity"),
		"user-agent":      m.client.GetHeader("user-agent"),
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if api.GetHTTPCode(result) == 200 {
		cookies := resp.Cookies()
		var sfCommunity, sessionApp string
		for _, c := range cookies {
			if c.Name == ".SFCommunity" {
				sfCommunity = c.Value
			}
			if c.Name == "session_APP" {
				sessionApp = c.Value
			}
		}
		return fmt.Sprintf(".SFCommunity=%s; session_APP=%s", sfCommunity, sessionApp), nil
	}

	return "", fmt.Errorf("登录失败")
}

// CheckLogin 检查当前登录状态
func (m *Manager) CheckLogin() bool {
	m.RefreshSecurityHeader()
	resp, err := m.client.Request("GET", api.BuildURL("/user?"), nil)
	if err != nil {
		return true
	}
	return api.GetHTTPCode(resp) != 200
}

// SetCookie 设置 cookie 到请求头
func (m *Manager) SetCookie(cookie string) {
	m.client.SetHeader("cookie", cookie)
}

// GetDeviceToken 返回设备令牌
func GetDeviceToken() string {
	return DeviceToken
}
