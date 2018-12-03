package config

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/banzaicloud/cicd-go/cicd"

	"github.com/99designs/httpsignatures-go"
)

func TestHandler(t *testing.T) {
	key := "xVKAGlWQiY3sOp8JVc0nbuNId3PNCgWh"

	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(&Request{})

	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", buf)
	req.Header.Add("Date", time.Now().UTC().Format(http.TimeFormat))

	err := httpsignatures.DefaultSha256Signer.AuthRequest("hmac-key", key, req)
	if err != nil {
		t.Error(err)
		return
	}

	want := &cicd.Config{
		Kind: "cicd.v1.yaml",
		Data: "pipeline: []",
	}
	plugin := &mockPlugin{
		res: want,
		err: nil,
	}

	handler := Handler(plugin, key, nil)
	handler.ServeHTTP(res, req)

	if got, want := res.Code, 200; got != want {
		t.Errorf("Want status code %d, got %d", want, got)
	}

	resp := &cicd.Config{}
	json.Unmarshal(res.Body.Bytes(), resp)
	if got, want := resp.Data, want.Data; got != want {
		t.Errorf("Want configuration data %s, got %s", want, got)
	}
}

func TestHandler_MissingSignature(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	handler := Handler(nil, "xVKAGlWQiY3sOp8JVc0nbuNId3PNCgWh", nil)
	handler.ServeHTTP(res, req)

	got, want := res.Body.String(), "Invalid or Missing Signature\n"
	if got != want {
		t.Errorf("Want response body %q, got %q", want, got)
	}
}

func TestHandler_InvalidSignature(t *testing.T) {
	sig := `keyId="hmac-key",algorithm="hmac-sha256",signature="QrS16+RlRsFjXn5IVW8tWz+3ZRAypjpNgzehEuvJksk=",headers="(request-target) accept accept-encoding content-type date digest"`
	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Signature", sig)

	handler := Handler(nil, "xVKAGlWQiY3sOp8JVc0nbuNId3PNCgWh", nil)
	handler.ServeHTTP(res, req)

	got, want := res.Body.String(), "Invalid Signature\n"
	if got != want {
		t.Errorf("Want response body %q, got %q", want, got)
	}
}

type mockPlugin struct {
	res *cicd.Config
	err error
}

func (m *mockPlugin) Find(ctx context.Context, req *Request) (*cicd.Config, error) {
	return m.res, m.err
}
