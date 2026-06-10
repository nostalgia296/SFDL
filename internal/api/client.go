// Package api 提供 HTTP 客户端和 API 请求功能
package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	BaseURL = "https://api.sfacg.com"
)

// Client 封装 HTTP 客户端和请求头
type Client struct {
	httpClient *http.Client
	headers    map[string]string
}

// NewClient 创建新的 API 客户端
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		headers: map[string]string{
			"Host":            "api.sfacg.com",
			"accept-charset":  "UTF-8",
			"authorization":   "Basic YW5kcm9pZHVzZXI6MWEjJDUxLXl0Njk7KkFjdkBxeHE=",
			"accept":          "application/vnd.sfacg.api+json;version=1",
			"accept-encoding": "gzip",
			"Content-Type":    "application/json; charset=UTF-8",
		},
	}
}

// SetHeader 设置请求头
func (c *Client) SetHeader(key, value string) {
	c.headers[key] = value
}

// GetHeader 获取请求头
func (c *Client) GetHeader(key string) string {
	return c.headers[key]
}

// Request 发送 HTTP 请求并返回解析后的 JSON 数据
func (c *Client) Request(method, url string, body []byte) (map[string]interface{}, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var reader io.ReadCloser
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, _ = gzip.NewReader(resp.Body)
		defer reader.Close()
	} else {
		reader = resp.Body
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetHTTPCode 从响应中提取 HTTP 状态码
func GetHTTPCode(resp map[string]interface{}) int {
	if status, ok := resp["status"].(map[string]interface{}); ok {
		if code, ok := status["httpCode"].(float64); ok {
			return int(code)
		}
	}
	return 0
}

// GetString 从 map 中安全获取字符串值
func GetString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// BuildURL 构建完整的 API URL
func BuildURL(path string) string {
	return fmt.Sprintf("%s%s", BaseURL, path)
}
