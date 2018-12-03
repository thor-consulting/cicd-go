package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/banzaicloud/cicd-go/cicd"
	"github.com/banzaicloud/cicd-go/plugin/internal/aesgcm"

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

	want := []*cicd.Registry{
		{
			Username: "docker_password",
			Password: "correct-horse-battery-staple",
		},
	}
	plugin := &mockPlugin{
		res: want,
		err: nil,
	}

	handler := Handler(key, plugin, nil)
	handler.ServeHTTP(res, req)

	if got, want := res.Code, 200; got != want {
		t.Errorf("Want status code %d, got %d", want, got)
	}

	resp := []*cicd.Registry{}
	json.Unmarshal(res.Body.Bytes(), &resp)
	if got, want := len(resp), len(want); got != want {
		t.Errorf("Want %d registry credentials, got %d", want, got)
		return
	}
	if got, want := resp[0].Username, want[0].Username; got != want {
		t.Errorf("Want registry username %s, got %s", want, got)
	}
	if got, want := resp[0].Password, want[0].Password; got != want {
		t.Errorf("Want registry password %s, got %s", want, got)
	}
}

func TestHandler_Encrypted(t *testing.T) {
	key := "xVKAGlWQiY3sOp8JVc0nbuNId3PNCgWh"

	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(&Request{})

	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", buf)
	req.Header.Add("Date", time.Now().UTC().Format(http.TimeFormat))
	req.Header.Add("Accept-Encoding", "aesgcm")

	err := httpsignatures.DefaultSha256Signer.AuthRequest("hmac-key", key, req)
	if err != nil {
		t.Error(err)
		return
	}

	want := []*cicd.Registry{
		{
			Username: "docker_password",
			Password: "correct-horse-battery-staple",
		},
	}
	plugin := &mockPlugin{
		res: want,
		err: nil,
	}

	handler := Handler(key, plugin, nil)
	handler.ServeHTTP(res, req)

	if got, want := res.Code, 200; got != want {
		t.Errorf("Want status code %d, got %d", want, got)
	}
	if got, want := res.Header().Get("Content-Encoding"), "aesgcm"; got != want {
		t.Errorf("Want Content-Encoding %s, got %s", want, got)
	}
	if got, want := res.Header().Get("Content-Type"), "application/octet-stream"; got != want {
		t.Errorf("Want Content-Type %s, got %s", want, got)
	}

	keyb, err := aesgcm.Key(key)
	if err != nil {
		t.Error(err)
		return
	}
	body, err := aesgcm.Decrypt(res.Body.Bytes(), keyb)
	if err != nil {
		t.Error(err)
		return
	}

	resp := []*cicd.Registry{}
	json.Unmarshal(body, &resp)
	if got, want := len(resp), len(want); got != want {
		t.Errorf("Want %d registry credentials, got %d", want, got)
		t.Errorf("Response body %s", res.Body)
		return
	}
	if got, want := resp[0].Username, want[0].Username; got != want {
		t.Errorf("Want registry username %s, got %s", want, got)
	}
	if got, want := resp[0].Password, want[0].Password; got != want {
		t.Errorf("Want registry password %s, got %s", want, got)
	}
}

func TestHandler_MissingSignature(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	handler := Handler("xVKAGlWQiY3sOp8JVc0nbuNId3PNCgWh", nil, nil)
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

	handler := Handler("xVKAGlWQiY3sOp8JVc0nbuNId3PNCgWh", nil, nil)
	handler.ServeHTTP(res, req)

	got, want := res.Body.String(), "Invalid Signature\n"
	if got != want {
		t.Errorf("Want response body %q, got %q", want, got)
	}
}

type mockPlugin struct {
	res []*cicd.Registry
	err error
}

func (m *mockPlugin) List(ctx context.Context, req *Request) ([]*cicd.Registry, error) {
	return m.res, m.err
}
