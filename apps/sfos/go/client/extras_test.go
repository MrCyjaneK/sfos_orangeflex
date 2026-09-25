package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestLoadListFAQ(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/faq", func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "brand=FLEX") || !strings.Contains(r.URL.RawQuery, "state=PUBLISHED") {
			t.Errorf("query %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"categoryName":{"en":"Technical stuff","pl":"X"},"articles":[{"title":{"en":"Watch eSIM"},"content":{"en":"Check Bluetooth."}}]}]`)
	})
	c := testClient(t, mux)
	if err := c.LoadList(context.Background(), "faq"); err != nil {
		t.Fatal(err)
	}
	items := c.ListItems("faq")
	if len(items) != 1 || items[0]["title"] != "Technical stuff" {
		t.Fatalf("%#v", items)
	}
	arts, _ := items[0]["articles"].([]map[string]any)
	if len(arts) != 1 || arts[0]["title"] != "Watch eSIM" {
		t.Fatalf("articles %#v", items[0]["articles"])
	}
}

func TestPaymentHistoryCapturedBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/payment/history", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("version") != "2" {
			t.Errorf("version %q", r.Header.Get("version"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ids, _ := body["payerId"].([]any)
		if len(ids) != 1 || ids[0] != "user-1" {
			t.Errorf("payer %#v", body["payerId"])
		}
		if body["productOfferingId"] != "TopUpDigitalWallet" {
			t.Errorf("offering %#v", body["productOfferingId"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"page":0,"perPage":5,"total":1,"payments":[{"status":"COMPLETED","transactionTimestamp":"2026-05-05T13:59:24.604Z","totalAmount":{"amount":"10.00","currency":"pln"},"paymentItem":[{"name":"TopUp (Digital Wallet)"}],"paymentMethod":{"methodType":"BLIK"}}]}`)
	})
	c := testClient(t, mux)
	c.lastDash = &Dashboard{User: User{ID: "user-1"}, Product: Product{ID: "prod-1"}}
	if err := c.LoadList(context.Background(), "payments"); err != nil {
		t.Fatal(err)
	}
	items := c.ListItems("payments")
	if len(items) != 1 || items[0]["amount"] != "10.00" || items[0]["method"] != "BLIK" {
		t.Fatalf("%#v", items)
	}
}

func TestTransferDataCapturedBody(t *testing.T) {
	var got map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/products/prod-1/balances/transfer", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(200)
	})
	c := testClient(t, mux)
	c.lastDash = &Dashboard{User: User{ID: "user-1"}, Product: Product{ID: "prod-1", Characteristic: map[string]any{"msisdn": "48500111222"}}}
	if err := c.TransferData(context.Background(), "500999888", "1", "hii"); err != nil {
		t.Fatal(err)
	}
	if got["type"] != "GB_TRANSFER" || got["targetMSISDN"] != "48500999888" || got["value"] != "1" || got["message"] != "hii" {
		t.Fatalf("%#v", got)
	}
	if got["checkMember"] != true {
		t.Fatalf("checkMember %#v", got["checkMember"])
	}
}

func TestTransferDataRejectsSelf(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/products/prod-1/balances/transfer", func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("must not POST self-transfer")
	})
	c := testClient(t, mux)
	c.lastDash = &Dashboard{User: User{ID: "user-1"}, Product: Product{ID: "prod-1", Characteristic: map[string]any{"msisdn": "48500111222"}}}
	if err := c.TransferData(context.Background(), "+48 500 111 222", "1", ""); err == nil {
		t.Fatal("expected self-transfer error")
	}
}

func TestTopUpWalletBLIKCaptured(t *testing.T) {
	var post map[string]any
	var gets int
	mux := http.NewServeMux()
	mux.HandleFunc("/payment", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		if r.URL.Path != "/payment" {
			return
		}
		if r.Header.Get("version") != "2" {
			t.Errorf("version %q", r.Header.Get("version"))
		}
		_ = json.NewDecoder(r.Body).Decode(&post)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_, _ = io.WriteString(w, `{"id":"pay-1","type":"PAYMENT","status":"PENDING"}`)
	})
	mux.HandleFunc("/payment/pay-1", func(w http.ResponseWriter, r *http.Request) {
		gets++
		if r.Header.Get("version") != "" {
			t.Errorf("GET version %q", r.Header.Get("version"))
		}
		if r.Header.Get("x-api-key") != APIKey {
			t.Errorf("x-api-key %q", r.Header.Get("x-api-key"))
		}
		if gets == 1 {
			if r.Header.Get("If-None-Match") != "" {
				t.Errorf("first poll sent If-None-Match")
			}
			w.Header().Set("ETag", `W/"abc"`)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"pay-1","status":"PENDING"}`)
			return
		}
		if r.Header.Get("If-None-Match") != `W/"abc"` {
			t.Errorf("etag %q", r.Header.Get("If-None-Match"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"pay-1","status":"COMPLETED"}`)
	})
	c := testClient(t, mux)
	c.pollInterval = time.Millisecond
	c.pollTries = 5
	c.lastDash = &Dashboard{User: User{ID: "user-1"}, Product: Product{ID: "prod-1"}}
	if err := c.TopUpWalletBLIK(context.Background(), "10", "318-137"); err != nil {
		t.Fatal(err)
	}
	if post["type"] != "PAYMENT" {
		t.Fatalf("type %#v", post["type"])
	}
	method, _ := post["paymentMethod"].(map[string]any)
	if method["methodType"] != "BLIK" || method["action"] != "ONE_CLICK" || method["authCode"] != "318137" {
		t.Fatalf("method %#v", method)
	}
	item := post["paymentItem"].([]any)[0].(map[string]any)
	amt := item["amount"].(map[string]any)
	if item["productOfferingId"] != "TopUpDigitalWallet" || amt["amount"] != 10.0 || amt["currency"] != "pln" {
		t.Fatalf("item %#v", item)
	}
	if gets < 2 {
		t.Fatalf("polls %d", gets)
	}
}

func TestWithdrawDataSafePostsAmount(t *testing.T) {
	var got map[string]any
	var status int
	mux := http.NewServeMux()
	mux.HandleFunc("/products/prod-1/balances/transfer", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		status = 201
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(201)
	})
	c := testClient(t, mux)
	c.lastDash = &Dashboard{User: User{ID: "user-1"}, Product: Product{ID: "prod-1"}, DataSafe: DataSafe{Amount: 10}}
	if err := c.WithdrawDataSafe(context.Background(), "1"); err != nil {
		t.Fatal(err)
	}
	if got["type"] != "BANK_WITHDRAWAL" || got["value"] != "1" {
		t.Fatalf("%#v", got)
	}
	if _, ok := got["targetMSISDN"]; ok {
		t.Fatalf("unexpected fields %#v", got)
	}
	if status != 201 {
		t.Fatalf("status %d", status)
	}
}

func TestWithdrawDataSafeRejectsOverdraw(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/products/prod-1/balances/transfer", func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("must not POST overdraw")
	})
	c := testClient(t, mux)
	c.lastDash = &Dashboard{User: User{ID: "user-1"}, Product: Product{ID: "prod-1"}, DataSafe: DataSafe{Amount: 1.5}}
	if err := c.WithdrawDataSafe(context.Background(), "2"); err == nil {
		t.Fatal("expected overdraw error")
	}
}

func TestExtraFromDashboard(t *testing.T) {
	d := &Dashboard{
		User: User{PaymentMethod: []PaymentMethod{{MethodType: "DIGITAL_WALLET", Active: true, Details: PaymentMethodDetails{Amount: "5.0000"}}}},
		Product: Product{
			Characteristic: map[string]any{"roaming": "Y", "clip": "Y"},
			Balances:       []ProductBalance{{Type: "METER_SOCIAL", Name: "Social", CurrentValue: "0.0", StartingValue: "infinity"}},
		},
		Counters: []Counter{{Type: "dataFUPCounters", Unit: "MB", CurrentValue: 1024, GrantValue: 2048}},
		DataSafe: DataSafe{Amount: 10, BalancePeriods: []DataSafePeriod{{CurrentValue: 4.5, ExpiryDate: "2026-10-10T06:26:08.000Z"}}},
	}
	ex := ExtraFromDashboard(d)
	if ex.RoamingLeft != "1" || ex.RoamingGrant != "2" {
		t.Fatalf("roaming %#v", ex)
	}
	if len(ex.Periods) != 1 || ex.Periods[0]["amount"] != "4.50" {
		t.Fatalf("periods %#v", ex.Periods)
	}
	if len(ex.Meters) != 1 || ex.Meters[0]["name"] != "Social" {
		t.Fatalf("meters %#v", ex.Meters)
	}
	if len(ex.Methods) != 1 || ex.Methods[0]["detail"] != "5.0000" {
		t.Fatalf("methods %#v", ex.Methods)
	}
}

func TestParseMessageContent(t *testing.T) {
	raw := `{"headings":{"en":"Stashed","pl":"Skrytka"},"contents":{"en":"Check Data Safe."}}`
	title, body := parseMessageContent(raw, "en")
	if title != "Stashed" || body != "Check Data Safe." {
		t.Fatalf("%q %q", title, body)
	}
}

func TestUnknownList(t *testing.T) {
	c := testClient(t, http.NewServeMux())
	if err := c.LoadList(context.Background(), "nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestMarkMessagesReadQuery(t *testing.T) {
	var rawQuery string
	mux := http.NewServeMux()
	mux.HandleFunc("/users/user-1/messages", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"messages":[{"id":11,"read":false,"content":"{\"headings\":{\"en\":\"A\"},\"contents\":{\"en\":\"B\"}}","date":"2026-09-10T06:35:48.201Z"}]}`)
	})
	mux.HandleFunc("/messages/read", func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		w.WriteHeader(204)
	})
	c := testClient(t, mux)
	c.lastDash = &Dashboard{User: User{ID: "user-1"}}
	if err := c.LoadList(context.Background(), "messages"); err != nil {
		t.Fatal(err)
	}
	if err := c.MarkMessagesRead(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rawQuery, "messageIds=11") {
		t.Fatalf("query %q", rawQuery)
	}
}
