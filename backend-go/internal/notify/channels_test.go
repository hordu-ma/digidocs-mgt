package notify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebhookPayloadShapes(t *testing.T) {
	for _, kind := range []string{"wecom", "dingtalk", "generic"} {
		var got map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &got)
		}))
		ch := WebhookChannel{URL: srv.URL, Kind: kind}
		err := ch.Deliver(context.Background(), Message{Title: "转交提醒", Body: "张三把《开题报告》转交给你", Link: "http://x/documents/1", Recipients: []Recipient{{DisplayName: "李四"}}})
		srv.Close()
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		switch kind {
		case "wecom", "dingtalk":
			md, _ := got["markdown"].(map[string]any)
			text, _ := md["content"].(string)
			if kind == "dingtalk" {
				text, _ = md["text"].(string)
			}
			if got["msgtype"] != "markdown" || !strings.Contains(text, "李四") || !strings.Contains(text, "http://x/documents/1") {
				t.Fatalf("%s payload: %#v", kind, got)
			}
		default:
			if got["title"] != "转交提醒" {
				t.Fatalf("generic payload: %#v", got)
			}
		}
	}
}

func TestWebhookErrorDoesNotLeakURL(t *testing.T) {
	ch := WebhookChannel{URL: "http://127.0.0.1:1/cgi-bin/webhook/send?key=SECRETKEY", Kind: "wecom"}
	err := ch.Deliver(context.Background(), Message{Title: "t"})
	if err == nil || strings.Contains(err.Error(), "SECRETKEY") {
		t.Fatalf("expected redacted error, got %v", err)
	}
}

func TestWebhookNon2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer srv.Close()
	if err := (WebhookChannel{URL: srv.URL, Kind: "generic"}).Deliver(context.Background(), Message{Title: "t"}); err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestBuildMailEncodesAndBlocksHeaderInjection(t *testing.T) {
	raw := string(BuildMail("a@x", "b@x\r\nBcc: evil@x", "转交\r\n提醒", "正文"))
	if strings.Contains(raw, "\r\nBcc:") {
		t.Fatalf("header injection not stripped: %q", raw)
	}
	if !strings.Contains(raw, "=?UTF-8?B?"+base64.StdEncoding.EncodeToString([]byte("转交提醒"))+"?=") {
		t.Fatalf("subject not encoded: %q", raw)
	}
}

func TestSMTPSkipsRecipientsWithoutEmail(t *testing.T) {
	ch := SMTPChannel{Host: "127.0.0.1", Port: 1, From: "a@x"}
	if err := ch.Deliver(context.Background(), Message{Recipients: []Recipient{{DisplayName: "无邮箱"}}}); err != nil {
		t.Fatalf("recipient without email must be skipped, got %v", err)
	}
}
