package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	dir := t.TempDir()
	SetNetworkAllowed(true)
	c, err := New(
		WithCacheDir(dir),
		WithHTTPClient(srv.Client()),
		WithGateways(srv.URL, srv.URL),
	)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLoginStartAndOTPThenDashboard(t *testing.T) {
	var sawAnalytics bool
	mux := http.NewServeMux()
	mux.HandleFunc("/verification/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("apikey") != APIKey {
			t.Errorf("missing apikey")
		}
		if r.Header.Get("User-Agent") != UserAgent {
			t.Errorf("ua %q", r.Header.Get("User-Agent"))
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		if body["channel"] == "EMAIL" {
			if body["to"] != "ticket-2" {
				t.Errorf("email start %#v", body)
			}
			_, _ = w.Write([]byte(`{"tid":"ticket-email","timeout":59}`))
			return
		}
		if body["to"] != "48500111222" {
			t.Errorf("to %q", body["to"])
		}
		_, _ = w.Write([]byte(`{"tid":"ticket-1","timeout":59,"expiresAt":"2026-09-22T07:20:13.966Z"}`))
	})
	mux.HandleFunc("/verification/action-token", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		switch body["otp"] {
		case "123456":
			if body["tid"] != "ticket-1" {
				t.Errorf("otp body %#v", body)
			}
			_, _ = w.Write([]byte(`{"tid":"ticket-2","masked_email":"c***@example.com"}`))
		case "654321":
			if body["tid"] != "ticket-email" {
				t.Errorf("email otp body %#v", body)
			}
			_, _ = w.Write([]byte(`{"action_token":"eyJhbGciOiJub25lIn0.e30.sig","device_id_token":"eyJhbGciOiJub25lIn0.e30.dev","tid":"ticket-email"}`))
		default:
			t.Errorf("unexpected otp %#v", body)
			w.WriteHeader(403)
		}
	})
	mux.HandleFunc("/proxy/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-build-number") != "" {
			t.Errorf("proxy/token must not send x-build-number")
		}
		_ = r.ParseForm()
		if r.Form.Get("subject_token") != "eyJhbGciOiJub25lIn0.e30.sig" {
			t.Errorf("subject %q", r.Form.Get("subject_token"))
		}
		if r.Form.Get("subject_token_type") != tokenJWTType {
			t.Errorf("type %q", r.Form.Get("subject_token_type"))
		}
		if r.Form.Get("grant_type") != tokenExchangeGrant {
			t.Errorf("grant %q", r.Form.Get("grant_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access-1","refresh_token":"refresh-1","expires_in":7200,"refresh_expires_in":1000,"token_type":"Bearer"}`))
	})
	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			http.Error(w, "no auth", 401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"user-1","firstName":"Ada","lastName":"Lovelace","type":"CUSTOMER","enabled":true,"paymentMethod":[{"methodType":"DIGITAL_WALLET","active":true,"details":{"amount":"5.0000"}}],"person":{"firstName":"Ada","lastName":"Lovelace"}}`))
	})
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"prod-1","status":"Active","specificationId":"MOBILE_ACCESS","offeringName":"Plan renewal","anniversaryDate":"2026-10-10T06:26:08.000Z","characteristic":{"msisdn":"48500111222"}}]`))
	})
	mux.HandleFunc("/products/prod-1/counters", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"type":"mainCounters","unit":"MB","currentValue":75785.08453,"grantValue":76800.0,"startingValue":76800.0,"anniversaryDate":"2026-10-10T06:26:08.000Z"}]`))
	})
	mux.HandleFunc("/products/prod-1/bank", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"amount":10.5,"balancePeriods":[]}`))
	})
	mux.HandleFunc("/groups", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"g1","type":"FAMILY","members":[{"id":"user-1","alias":"ADA","role":"ADMIN"}]}]`))
	})
	mux.HandleFunc("/SIM/list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"parentProductId":"prod-1","simCards":[{"cardHierarchy":"Primary","cardType":"ESIM"}]}`))
	})
	mux.HandleFunc("/api/v6.18/androidevent", func(w http.ResponseWriter, r *http.Request) {
		sawAnalytics = true
		w.WriteHeader(200)
	})

	c := testClient(t, mux)
	ctx := context.Background()
	st, err := c.LoginStart(ctx, "48500111222")
	if err != nil {
		t.Fatal(err)
	}
	if st.TID != "ticket-1" || st.Timeout != 59 {
		t.Fatalf("start %#v", st)
	}
	otp, err := c.LoginSubmitOTP(ctx, "123456")
	if err != nil {
		t.Fatal(err)
	}
	if !otp.NeedNextOTP || otp.MaskedEmail != "c***@example.com" {
		t.Fatalf("otp %#v", otp)
	}
	otp, err = c.LoginSubmitOTP(ctx, "654321")
	if err != nil {
		t.Fatal(err)
	}
	if !otp.LoggedIn {
		t.Fatalf("email otp %#v", otp)
	}
	if !c.LoggedIn() {
		t.Fatal("expected tokens")
	}
	d, err := c.LoadDashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d.DisplayName() != "Ada" {
		t.Fatalf("name %q", d.DisplayName())
	}
	left, grant := CounterGB(d.MainCounter())
	if grant != 75 {
		t.Fatalf("grant %v", grant)
	}
	if left < 74 || left > 75 {
		t.Fatalf("left %v", left)
	}
	if d.SIMs.SIMCards[0].CardType != "ESIM" {
		t.Fatalf("sim %#v", d.SIMs)
	}
	if sawAnalytics {
		t.Fatal("must not call analytics endpoints")
	}
	if _, err := os.Stat(filepath.Join(c.CacheDir(), "session.json")); err != nil {
		t.Fatal("session not persisted")
	}
}

func TestLoginStartDeniedWithoutNetwork(t *testing.T) {
	SetNetworkAllowed(false)
	c, err := New(WithCacheDir(t.TempDir()), WithGateways("http://127.0.0.1:1", "http://127.0.0.1:1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.LoginStart(context.Background(), "48500")
	if err == nil || !strings.Contains(err.Error(), "network permission") {
		t.Fatalf("err %v", err)
	}
}

func TestExistingUserRestartsSMSAfterRegistrationToken(t *testing.T) {
	smsStarts := 0
	tokenExchanges := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/verification/start", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		if body["channel"] == "EMAIL" {
			if body["to"] != "ticket-sms" {
				t.Errorf("email start %#v", body)
			}
			_, _ = w.Write([]byte(`{"tid":"ticket-email","timeout":59}`))
			return
		}
		smsStarts++
		switch smsStarts {
		case 1:
			_, _ = w.Write([]byte(`{"tid":"ticket-sms","timeout":59}`))
		case 2:
			_, _ = w.Write([]byte(`{"tid":"ticket-retry","timeout":45}`))
		default:
			t.Errorf("unexpected sms start %d %#v", smsStarts, body)
			w.WriteHeader(500)
		}
	})
	mux.HandleFunc("/verification/action-token", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		switch body["tid"] {
		case "ticket-sms":
			_, _ = w.Write([]byte(`{"tid":"ticket-sms","masked_email":"c***@example.com"}`))
		case "ticket-email":
			_, _ = w.Write([]byte(`{"action_token":"eyJhbGciOiJub25lIn0.e30.reg","device_id_token":"eyJhbGciOiJub25lIn0.e30.dev","tid":"ticket-email"}`))
		case "ticket-retry":
			if body["device_id_token"] != "eyJhbGciOiJub25lIn0.e30.dev" {
				t.Errorf("retry otp missing device_id_token %#v", body)
			}
			_, _ = w.Write([]byte(`{"action_token":"eyJhbGciOiJub25lIn0.e30.login","tid":"ticket-retry"}`))
		default:
			t.Errorf("unexpected tid %#v", body)
			w.WriteHeader(403)
		}
	})
	mux.HandleFunc("/proxy/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		tokenExchanges++
		w.Header().Set("Content-Type", "application/json")
		switch r.Form.Get("subject_token") {
		case "eyJhbGciOiJub25lIn0.e30.reg":
			w.WriteHeader(400)
			_, _ = io.WriteString(w, `{"error":"invalid_token","error_description":"User already exists"}`)
		case "eyJhbGciOiJub25lIn0.e30.login":
			_, _ = io.WriteString(w, `{"access_token":"access-2","refresh_token":"r2","expires_in":60}`)
		default:
			t.Errorf("subject %q", r.Form.Get("subject_token"))
			w.WriteHeader(400)
		}
	})
	c := testClient(t, mux)
	ctx := context.Background()
	if _, err := c.LoginStart(ctx, "48500111222"); err != nil {
		t.Fatal(err)
	}
	otp, err := c.LoginSubmitOTP(ctx, "111111")
	if err != nil {
		t.Fatal(err)
	}
	if !otp.NeedNextOTP {
		t.Fatalf("sms otp %#v", otp)
	}
	otp, err = c.LoginSubmitOTP(ctx, "222222")
	if err != nil {
		t.Fatal(err)
	}
	if !otp.RestartSMS || otp.LoggedIn || otp.Timeout != 45 {
		t.Fatalf("email otp %#v", otp)
	}
	if c.PendingTID() != "ticket-retry" {
		t.Fatalf("pending %q", c.PendingTID())
	}
	if c.deviceIDToken() != "eyJhbGciOiJub25lIn0.e30.dev" {
		t.Fatalf("device token %q", c.deviceIDToken())
	}
	otp, err = c.LoginSubmitOTP(ctx, "333333")
	if err != nil {
		t.Fatal(err)
	}
	if !otp.LoggedIn {
		t.Fatalf("retry otp %#v", otp)
	}
	if !c.LoggedIn() {
		t.Fatal("expected tokens")
	}
	if smsStarts != 2 || tokenExchanges != 2 {
		t.Fatalf("smsStarts=%d tokenExchanges=%d", smsStarts, tokenExchanges)
	}
}

func TestActionTokenDoesNotResubmitConsumedOTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/verification/start", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"tid":"t1","timeout":30}`)
	})
	mux.HandleFunc("/verification/action-token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"action_token":"eyJhbGciOiJub25lIn0.e30.sig","tid":"t1"}`))
	})
	mux.HandleFunc("/proxy/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = io.WriteString(w, `{"error":"invalid_token","error_description":"nope"}`)
	})
	c := testClient(t, mux)
	if _, err := c.LoginStart(context.Background(), "48"); err != nil {
		t.Fatal(err)
	}
	_, err := c.LoginSubmitOTP(context.Background(), "111111")
	if err == nil {
		t.Fatal("expected exchange error")
	}
	if c.PendingTID() != "" {
		t.Fatalf("consumed ticket still pending %q", c.PendingTID())
	}
}

func TestTrustedDeviceOTPExchangesActionToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/verification/start", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"tid":"t1","timeout":30}`)
	})
	mux.HandleFunc("/verification/action-token", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["device_id_token"] != "eyJhbGciOiJub25lIn0.e30.dev" {
			t.Errorf("missing device_id_token %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"action_token":"eyJhbGciOiJub25lIn0.e30.sig","tid":"t1"}`))
	})
	mux.HandleFunc("/proxy/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("subject_token") != "eyJhbGciOiJub25lIn0.e30.sig" {
			t.Errorf("subject %q", r.Form.Get("subject_token"))
		}
		if r.Form.Get("subject_token_type") != tokenJWTType {
			t.Errorf("type %q", r.Form.Get("subject_token_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"access-jwt","refresh_token":"r","expires_in":60}`)
	})
	c := testClient(t, mux)
	c.setDeviceIDToken("eyJhbGciOiJub25lIn0.e30.dev")
	if _, err := c.LoginStart(context.Background(), "48"); err != nil {
		t.Fatal(err)
	}
	res, err := c.LoginSubmitOTP(context.Background(), "111111")
	if err != nil {
		t.Fatal(err)
	}
	if !res.LoggedIn {
		t.Fatalf("res %#v", res)
	}
}

func TestInvalidOTP(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/verification/start", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"tid":"t1","timeout":30}`)
	})
	mux.HandleFunc("/verification/action-token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(403)
		_, _ = io.WriteString(w, `{"code":"50019","message":"Invalid OTP","timeout":60}`)
	})
	c := testClient(t, mux)
	if _, err := c.LoginStart(context.Background(), "48"); err != nil {
		t.Fatal(err)
	}
	_, err := c.LoginSubmitOTP(context.Background(), "000000")
	if err == nil {
		t.Fatal("expected error")
	}
	ae, ok := err.(*APIError)
	if !ok || ae.Status != 403 || ae.Message != "Invalid OTP" {
		t.Fatalf("got %#v", err)
	}
}

func TestNoAnalyticsInClientSource(t *testing.T) {
	root := filepath.Join("..")
	err := filepath.Walk(filepath.Join(root, "client"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		s := string(b)
		for _, needle := range []string{"appsflyer", "onesignal", "codepush", "bugsnag", "firebase", "androidevent"} {
			if strings.Contains(strings.ToLower(s), needle) {
				t.Errorf("%s mentions %s", path, needle)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
