package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	CfgDingEnabled = "notify_ding_enabled"
	CfgDingWebhook = "notify_ding_webhook"
	CfgDingSecret  = "notify_ding_secret"
	CfgWxEnabled   = "notify_wx_enabled"
	CfgWxWebhook   = "notify_wx_webhook"
	CfgNotifyAt    = "notify_at"
)

func (a *App) NotifyFailure(taskName, vendorName, errMsg string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	content := fmt.Sprintf(
		"【数据交换任务执行失败】\n时间: %s\n厂家: %s\n任务: %s\n错误: %s",
		now, vendorName, taskName, errMsg,
	)

	if a.GetConfigWithDefault(CfgDingEnabled, "off") == "on" {
		if wb := a.GetConfig(CfgDingWebhook); wb != "" {
			if err := sendDingTalk(wb, a.GetConfig(CfgDingSecret), content); err != nil {
				log.Printf("[通知] 钉钉发送失败: %v", err)
			} else {
				log.Printf("[通知] 钉钉提醒已发送 (任务: %s)", taskName)
			}
		}
	}

	if a.GetConfigWithDefault(CfgWxEnabled, "off") == "on" {
		if wb := a.GetConfig(CfgWxWebhook); wb != "" {
			if err := sendWeChat(wb, content, a.GetConfig(CfgNotifyAt)); err != nil {
				log.Printf("[通知] 企业微信发送失败: %v", err)
			} else {
				log.Printf("[通知] 企业微信提醒已发送 (任务: %s)", taskName)
			}
		}
	}
}

func dingSign(secret, timestamp string) string {
	data := timestamp + "\n" + secret
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func sendDingTalk(rawWebhook, secret, content string) error {
	webhook := rawWebhook
	if secret != "" {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		sign := dingSign(secret, timestamp)
		u, err := url.Parse(webhook)
		if err != nil {
			return fmt.Errorf("webhook解析失败: %v", err)
		}
		q := u.Query()
		q.Set("timestamp", timestamp)
		q.Set("sign", sign)
		u.RawQuery = q.Encode()
		webhook = u.String()
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": content},
	}
	return postJSON(webhook, payload)
}

func sendWeChat(webhook, content, at string) error {
	text := map[string]interface{}{"content": content}
	if at != "" {
		parts := splitAndTrim(at)
		if len(parts) > 0 {
			text["mentioned_list"] = parts
		}
	}
	payload := map[string]interface{}{
		"msgtype": "text",
		"text":    text,
	}
	return postJSON(webhook, payload)
}

func splitAndTrim(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func postJSON(webhook string, payload interface{}) error {
	body, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhook, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP状态异常: %d", resp.StatusCode)
	}

	var r struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if json.NewDecoder(resp.Body).Decode(&r) == nil && r.ErrCode != 0 {
		return fmt.Errorf("平台返回错误: errcode=%d, errmsg=%s", r.ErrCode, r.ErrMsg)
	}
	return nil
}
