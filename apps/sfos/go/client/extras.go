package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Localized is locize-style en/pl/uk copy.
type Localized struct {
	EN string `json:"en"`
	PL string `json:"pl"`
	UK string `json:"uk"`
}

// Pick chooses copy for the UI language.
func (l Localized) Pick(lang string) string {
	switch strings.ToLower(lang) {
	case "pl":
		if l.PL != "" {
			return l.PL
		}
	case "uk":
		if l.UK != "" {
			return l.UK
		}
	}
	if l.EN != "" {
		return l.EN
	}
	if l.PL != "" {
		return l.PL
	}
	return l.UK
}

func pickMap(m map[string]string, lang string) string {
	if m == nil {
		return ""
	}
	lang = strings.ToLower(lang)
	if s := m[lang]; s != "" {
		return s
	}
	if s := m["en"]; s != "" {
		return s
	}
	if s := m["pl"]; s != "" {
		return s
	}
	return m["uk"]
}

func row(kv ...string) map[string]any {
	out := map[string]any{}
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i]] = kv[i+1]
	}
	return out
}

func (c *Client) storeList(name string, items []map[string]any) {
	if items == nil {
		items = []map[string]any{}
	}
	c.mu.Lock()
	if c.lists == nil {
		c.lists = map[string][]map[string]any{}
	}
	c.lists[name] = items
	c.mu.Unlock()
}

// ListItems is the last snapshot for name.
func (c *Client) ListItems(name string) []map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lists == nil {
		return nil
	}
	return c.lists[name]
}

func (c *Client) lastDashboard() *Dashboard {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastDash
}

func (c *Client) dashIDs() (userID, productID string) {
	d := c.lastDashboard()
	if d == nil {
		c.mu.Lock()
		userID = c.session.UserID
		c.mu.Unlock()
		return userID, ""
	}
	return d.User.ID, d.Product.ID
}

// DashboardExtra is home / my-number fields beyond the original C scalars.
type DashboardExtra struct {
	RoamingLeft  string           `json:"roamingLeft"`
	RoamingGrant string           `json:"roamingGrant"`
	Periods      []map[string]any `json:"periods"`
	Meters       []map[string]any `json:"meters"`
	Methods      []map[string]any `json:"methods"`
	Flags        []map[string]any `json:"flags"`
}

// ExtraFromDashboard flattens counters, Data Safe periods, meters, and flags.
func ExtraFromDashboard(d *Dashboard) DashboardExtra {
	ex := DashboardExtra{}
	if d == nil {
		return ex
	}
	left, grant := CounterGB(d.RoamingCounter())
	if d.RoamingCounter().Type != "" {
		ex.RoamingLeft = FormatGB(left)
		ex.RoamingGrant = FormatGB(grant)
	}
	for _, p := range d.DataSafe.BalancePeriods {
		ex.Periods = append(ex.Periods, row(
			"amount", FormatGB(p.CurrentValue),
			"activation", FormatRenewal(p.ActivationDate),
			"expiry", FormatRenewal(p.ExpiryDate),
		))
	}
	for _, b := range d.Product.Balances {
		ex.Meters = append(ex.Meters, row(
			"name", b.Name,
			"type", b.Type,
			"value", fmt.Sprint(b.CurrentValue),
			"start", fmt.Sprint(b.StartingValue),
			"unit", b.Unit,
		))
	}
	for _, pm := range d.User.PaymentMethod {
		detail := pm.Details.Amount
		if pm.Details.MaskedCardNumber != "" {
			detail = pm.Details.Issuer + " " + pm.Details.MaskedCardNumber
		}
		ex.Methods = append(ex.Methods, row(
			"type", pm.MethodType,
			"detail", strings.TrimSpace(detail),
			"active", strconv.FormatBool(pm.Active),
		))
	}
	keys := []string{"roaming", "clip", "clir", "callWaiting", "callHold", "mms", "smsOutgoing", "smsIncoming"}
	for _, k := range keys {
		if d.Product.Characteristic == nil {
			break
		}
		if v, ok := d.Product.Characteristic[k]; ok {
			ex.Flags = append(ex.Flags, row("name", k, "value", fmt.Sprint(v)))
		}
	}
	return ex
}

// ExtraJSON is dashboard extras for the Qt bridge.
func (c *Client) ExtraJSON() ([]byte, error) {
	ex := ExtraFromDashboard(c.lastDashboard())
	return json.Marshal(ex)
}

// LoadList fetches a named captured catalog and stores a flattened snapshot.
// Names: faq, messages, payments, invoices, orders, offers, roamingOffers,
// countries, consents, esims, documents, services.
func (c *Client) LoadList(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	var (
		items []map[string]any
		err   error
	)
	switch name {
	case "faq":
		items, err = c.fetchFAQ(ctx)
	case "messages":
		items, err = c.fetchMessages(ctx)
	case "payments":
		items, err = c.fetchPayments(ctx)
	case "invoices":
		items, err = c.fetchInvoices(ctx)
	case "orders":
		items, err = c.fetchOrders(ctx)
	case "offers":
		items, err = c.fetchProductOffers(ctx)
	case "roamingOffers":
		items, err = c.fetchRoamingOffers(ctx)
	case "countries":
		items, err = c.fetchCountries(ctx)
	case "consents":
		items, err = c.fetchConsents(ctx)
	case "esims":
		items, err = c.fetchESIMs(ctx)
	case "documents":
		items, err = c.fetchDocuments(ctx)
	case "services":
		items, err = c.fetchServices(ctx)
	default:
		return fmt.Errorf("client: unknown list %q", name)
	}
	if err != nil {
		return err
	}
	c.storeList(name, items)
	return nil
}

func (c *Client) fetchFAQ(ctx context.Context) ([]map[string]any, error) {
	c.report("help", "Loading FAQ")
	u, err := url.Parse(c.anonHost + "/faq")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("_sort", "priority:ASC")
	q.Set("state", "PUBLISHED")
	q.Set("displayMobile", "true")
	q.Add("brand", "FLEX")
	q.Add("brand", "FLEX_TRAVEL")
	u.RawQuery = q.Encode()
	var cats []struct {
		CategoryName Localized `json:"categoryName"`
		Articles     []struct {
			Title   Localized `json:"title"`
			Content Localized `json:"content"`
		} `json:"articles"`
	}
	if err := c.getJSON(ctx, u.String(), false, &cats); err != nil {
		return nil, err
	}
	lang := c.Language()
	out := make([]map[string]any, 0, len(cats))
	for _, cat := range cats {
		arts := make([]map[string]any, 0, len(cat.Articles))
		for _, a := range cat.Articles {
			arts = append(arts, map[string]any{
				"title":   a.Title.Pick(lang),
				"content": a.Content.Pick(lang),
			})
		}
		out = append(out, map[string]any{
			"title":    cat.CategoryName.Pick(lang),
			"articles": arts,
		})
	}
	return out, nil
}

func (c *Client) fetchMessages(ctx context.Context) ([]map[string]any, error) {
	c.report("inbox", "Loading messages")
	uid, _ := c.dashIDs()
	if uid == "" {
		return nil, fmt.Errorf("client: no user id")
	}
	raw := c.publicHost + "/users/" + url.PathEscape(uid) + "/messages?channel=PUSH"
	var page struct {
		Messages []struct {
			ID      json.Number `json:"id"`
			Date    string      `json:"date"`
			Read    bool        `json:"read"`
			Content string      `json:"content"`
		} `json:"messages"`
	}
	if err := c.getJSON(ctx, raw, true, &page); err != nil {
		return nil, err
	}
	lang := c.Language()
	out := make([]map[string]any, 0, len(page.Messages))
	unread := make([]string, 0)
	for _, m := range page.Messages {
		title, body := parseMessageContent(m.Content, lang)
		id := m.ID.String()
		out = append(out, row("id", id, "title", title, "body", body, "date", FormatRenewal(m.Date), "read", strconv.FormatBool(m.Read)))
		if !m.Read && id != "" {
			unread = append(unread, id)
		}
	}
	c.mu.Lock()
	c.unread = unread
	c.mu.Unlock()
	return out, nil
}

func parseMessageContent(raw, lang string) (title, body string) {
	var payload struct {
		Headings map[string]string `json:"headings"`
		Contents map[string]string `json:"contents"`
	}
	if json.Unmarshal([]byte(raw), &payload) != nil {
		return "", raw
	}
	return pickMap(payload.Headings, lang), pickMap(payload.Contents, lang)
}

// MarkMessagesRead is PATCH /messages/read?messageIds=… for unread inbox rows.
func (c *Client) MarkMessagesRead(ctx context.Context) error {
	c.mu.Lock()
	ids := append([]string(nil), c.unread...)
	c.mu.Unlock()
	if len(ids) == 0 {
		return nil
	}
	q := url.Values{}
	for _, id := range ids {
		q.Add("messageIds", id)
	}
	return c.patchEmpty(ctx, c.publicHost+"/messages/read?"+q.Encode(), true)
}

func (c *Client) fetchPayments(ctx context.Context) ([]map[string]any, error) {
	c.report("payments", "Loading payments")
	uid, _ := c.dashIDs()
	if uid == "" {
		return nil, fmt.Errorf("client: no user id")
	}
	payload := map[string]any{
		"payerId":           []string{uid},
		"type":              []string{"PAYMENT"},
		"productOfferingId": "TopUpDigitalWallet",
	}
	var page struct {
		Payments []struct {
			Status           string `json:"status"`
			TransactionStamp string `json:"transactionTimestamp"`
			TotalAmount      struct {
				Amount   string `json:"amount"`
				Currency string `json:"currency"`
			} `json:"totalAmount"`
			PaymentItem []struct {
				Name string `json:"name"`
			} `json:"paymentItem"`
			PaymentMethod struct {
				MethodType string `json:"methodType"`
			} `json:"paymentMethod"`
		} `json:"payments"`
	}
	if err := c.postJSONWithHeaders(ctx, c.publicHost+"/payment/history", true, payload, &page, map[string]string{"version": "2"}); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(page.Payments))
	for _, p := range page.Payments {
		name := "Payment"
		if len(p.PaymentItem) > 0 && p.PaymentItem[0].Name != "" {
			name = p.PaymentItem[0].Name
		}
		out = append(out, row(
			"name", name,
			"amount", p.TotalAmount.Amount,
			"currency", strings.ToUpper(p.TotalAmount.Currency),
			"status", p.Status,
			"date", FormatRenewal(p.TransactionStamp),
			"method", p.PaymentMethod.MethodType,
		))
	}
	return out, nil
}

func (c *Client) fetchInvoices(ctx context.Context) ([]map[string]any, error) {
	c.report("payments", "Loading invoices")
	uid, _ := c.dashIDs()
	if uid == "" {
		return nil, fmt.Errorf("client: no user id")
	}
	q := url.Values{}
	q.Set("userId", uid)
	q.Set("perPage", "15")
	q.Set("page", "0")
	q.Set("state[0]", "SENT")
	q.Set("state[1]", "BLOCKED")
	var page struct {
		Invoice []map[string]any `json:"invoice"`
	}
	if err := c.getJSON(ctx, c.publicHost+"/users/invoices?"+q.Encode(), true, &page); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(page.Invoice))
	for _, inv := range page.Invoice {
		out = append(out, row(
			"id", fmt.Sprint(inv["id"]),
			"state", fmt.Sprint(inv["state"]),
			"date", FormatRenewal(fmt.Sprint(inv["issueDate"])),
			"amount", fmt.Sprint(inv["grossAmount"]),
		))
	}
	return out, nil
}

func (c *Client) fetchOrders(ctx context.Context) ([]map[string]any, error) {
	c.report("orders", "Loading orders")
	var page struct {
		Orders []struct {
			ID        string `json:"id"`
			Status    string `json:"status"`
			Category  string `json:"category"`
			Sub       string `json:"subCategory"`
			Process   string `json:"processId"`
			OrderDate string `json:"orderDate"`
			Delivery  struct {
				DeliveryType    string `json:"deliveryType"`
				Municipality    string `json:"municipality"`
				ProviderPointID string `json:"providerPointId"`
				Note            string `json:"note"`
			} `json:"delivery"`
			OrderItem []struct {
				Product struct {
					Name string `json:"name"`
				} `json:"product"`
			} `json:"orderItem"`
		} `json:"orders"`
	}
	if err := c.getJSON(ctx, c.publicHost+"/orders", true, &page); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(page.Orders))
	for _, o := range page.Orders {
		name := o.Sub
		if name == "" {
			name = o.Process
		}
		if len(o.OrderItem) > 0 && o.OrderItem[0].Product.Name != "" {
			name = o.OrderItem[0].Product.Name
		}
		place := strings.TrimSpace(strings.Join([]string{o.Delivery.DeliveryType, o.Delivery.ProviderPointID, o.Delivery.Municipality}, " "))
		out = append(out, row(
			"id", o.ID,
			"name", name,
			"status", o.Status,
			"category", o.Category,
			"date", FormatRenewal(o.OrderDate),
			"delivery", strings.TrimSpace(place+" "+o.Delivery.Note),
		))
	}
	return out, nil
}

func (c *Client) fetchProductOffers(ctx context.Context) ([]map[string]any, error) {
	c.report("shop", "Loading data passes")
	uid, pid := c.dashIDs()
	if pid == "" {
		return nil, fmt.Errorf("client: no product id")
	}
	q := url.Values{}
	q.Set("active", "true")
	q.Set("category", "DATAPASS")
	if uid != "" {
		q.Set("userId", uid)
	}
	return c.flattenOffers(ctx, c.publicHost+"/products/"+url.PathEscape(pid)+"/offers?"+q.Encode(), true, "")
}

func (c *Client) fetchRoamingOffers(ctx context.Context) ([]map[string]any, error) {
	c.report("roaming", "Loading roaming packs")
	return c.flattenOffers(ctx, c.publicHost+"/offers?active=true", true, "ROAMING")
}

type offerRow struct {
	ID                   string `json:"id"`
	Category             string `json:"category"`
	Name                 string `json:"name"`
	RefillPeriod         int    `json:"refillPeriod"`
	Zone                 string `json:"zone"`
	ProductOfferingPrice struct {
		Price struct {
			Amount       float64 `json:"amount"`
			CurrencyCode string  `json:"currencyCode"`
		} `json:"price"`
	} `json:"productOfferingPrice"`
}

func (c *Client) flattenOffers(ctx context.Context, rawURL string, authed bool, category string) ([]map[string]any, error) {
	var offers []offerRow
	if err := c.getJSON(ctx, rawURL, authed, &offers); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(offers))
	for _, o := range offers {
		if category != "" && !strings.EqualFold(o.Category, category) {
			continue
		}
		price := strconv.FormatFloat(o.ProductOfferingPrice.Price.Amount, 'f', -1, 64)
		cur := o.ProductOfferingPrice.Price.CurrencyCode
		period := ""
		if o.RefillPeriod > 0 {
			period = strconv.Itoa(o.RefillPeriod)
		}
		out = append(out, row(
			"id", o.ID,
			"name", o.Name,
			"category", o.Category,
			"price", price,
			"currency", cur,
			"period", period,
			"zone", o.Zone,
		))
	}
	return out, nil
}

func (c *Client) fetchCountries(ctx context.Context) ([]map[string]any, error) {
	c.report("roaming", "Loading countries")
	q := url.Values{}
	q.Set("brand", "FLEX")
	q.Set("category", "ROAMING")
	var countries []struct {
		ISO         string    `json:"ISO_3166_1_alpha_2"`
		RoamingZone string    `json:"roamingZone"`
		DisplayName Localized `json:"displayName"`
		Zones       []struct {
			DisplayName Localized `json:"displayName"`
			Description Localized `json:"description"`
		} `json:"zones"`
	}
	if err := c.getJSON(ctx, c.anonHost+"/cms/countries?"+q.Encode(), false, &countries); err != nil {
		return nil, err
	}
	lang := c.Language()
	out := make([]map[string]any, 0, len(countries))
	for _, co := range countries {
		zone := co.RoamingZone
		desc := ""
		if len(co.Zones) > 0 {
			if n := co.Zones[0].DisplayName.Pick(lang); n != "" {
				zone = n
			}
			desc = co.Zones[0].Description.Pick(lang)
		}
		out = append(out, row(
			"name", co.DisplayName.Pick(lang),
			"iso", co.ISO,
			"zone", zone,
			"description", desc,
		))
	}
	return out, nil
}

func (c *Client) fetchConsents(ctx context.Context) ([]map[string]any, error) {
	c.report("settings", "Loading consents")
	uid, _ := c.dashIDs()
	if uid == "" {
		return nil, fmt.Errorf("client: no user id")
	}
	var consents []struct {
		ConsentTitle string `json:"consentTitle"`
		ConsentText  string `json:"consentText"`
		Accepted     bool   `json:"accepted"`
	}
	if err := c.getJSON(ctx, c.publicHost+"/users/"+url.PathEscape(uid)+"/consents", true, &consents); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(consents))
	for _, co := range consents {
		out = append(out, row(
			"title", co.ConsentTitle,
			"text", co.ConsentText,
			"accepted", strconv.FormatBool(co.Accepted),
		))
	}
	return out, nil
}

func (c *Client) fetchESIMs(ctx context.Context) ([]map[string]any, error) {
	c.report("sim", "Loading eSIMs")
	uid, _ := c.dashIDs()
	raw := c.publicHost + "/esims"
	if uid != "" {
		raw += "?userId=" + url.QueryEscape(uid)
	}
	var cards []struct {
		ICCID      string `json:"iccid"`
		State      string `json:"state"`
		MatchingID string `json:"matchingId"`
		SubState   string `json:"subState"`
	}
	if err := c.getJSON(ctx, raw, true, &cards); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(cards))
	for _, e := range cards {
		out = append(out, row(
			"iccid", e.ICCID,
			"state", e.State,
			"matchingId", e.MatchingID,
			"subState", e.SubState,
		))
	}
	return out, nil
}

func (c *Client) fetchDocuments(ctx context.Context) ([]map[string]any, error) {
	c.report("settings", "Loading documents")
	raw := c.anonHost + "/cms/documents?categoryId=" + url.QueryEscape("FLEX_APP_TERMS_AND_CONDITIONS")
	var docs []struct {
		Title Localized `json:"title"`
		URL   string    `json:"url"`
	}
	if err := c.getJSON(ctx, raw, false, &docs); err != nil {
		return nil, err
	}
	lang := c.Language()
	out := make([]map[string]any, 0, len(docs))
	for _, d := range docs {
		out = append(out, row("title", d.Title.Pick(lang), "url", d.URL))
	}
	return out, nil
}

func (c *Client) fetchServices(ctx context.Context) ([]map[string]any, error) {
	c.report("number", "Loading network services")
	var cats []struct {
		Name            string `json:"name"`
		NetworkServices []struct {
			Name                 string    `json:"name"`
			IsVisibleForCustomer bool      `json:"isVisibleForCustomer"`
			DisplayName          Localized `json:"displayName"`
		} `json:"networkServices"`
	}
	if err := c.getJSON(ctx, c.publicHost+"/cms/network-services-category", true, &cats); err != nil {
		return nil, err
	}
	var ch map[string]any
	if d := c.lastDashboard(); d != nil {
		ch = d.Product.Characteristic
	}
	lang := c.Language()
	out := []map[string]any{}
	for _, cat := range cats {
		for _, svc := range cat.NetworkServices {
			if !svc.IsVisibleForCustomer {
				continue
			}
			val := ""
			if ch != nil {
				if v, ok := ch[svc.Name]; ok {
					val = fmt.Sprint(v)
				}
			}
			out = append(out, row(
				"category", cat.Name,
				"name", svc.Name,
				"title", svc.DisplayName.Pick(lang),
				"value", val,
			))
		}
	}
	return out, nil
}

// TransferData is POST /products/{id}/balances/transfer as captured.
func (c *Client) TransferData(ctx context.Context, targetMSISDN, value, message string) error {
	_, pid := c.dashIDs()
	if pid == "" {
		return fmt.Errorf("client: no product id")
	}
	targetMSISDN = normalizeMSISDN(targetMSISDN)
	value = strings.TrimSpace(value)
	if targetMSISDN == "" || value == "" {
		return fmt.Errorf("client: transfer needs a number and amount")
	}
	if own := normalizeMSISDN(c.ownMSISDN()); own != "" && targetMSISDN == own {
		return fmt.Errorf("client: that is your own number")
	}
	c.report("transfer", "Sending data")
	payload := map[string]any{
		"type":         "GB_TRANSFER",
		"targetMSISDN": targetMSISDN,
		"value":        value,
		"checkMember":  true,
		"message":      message,
	}
	return c.postJSON(ctx, c.publicHost+"/products/"+url.PathEscape(pid)+"/balances/transfer", true, payload, nil)
}

func (c *Client) ownMSISDN() string {
	d := c.lastDashboard()
	if d == nil {
		return ""
	}
	if n := d.Product.MSISDN(); n != "" {
		return n
	}
	return ""
}

func normalizeMSISDN(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if strings.HasPrefix(out, "00") {
		out = out[2:]
	}
	if len(out) == 9 {
		out = "48" + out
	}
	return out
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func parsePositiveAmount(s string) (float64, error) {
	n, err := strconv.ParseFloat(strings.TrimSpace(strings.ReplaceAll(s, ",", ".")), 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("client: amount must be greater than zero")
	}
	return n, nil
}

func (c *Client) pollSettings() (wait time.Duration, tries int) {
	wait = 2 * time.Second
	tries = 30
	if c.pollInterval > 0 {
		wait = c.pollInterval
	}
	if c.pollTries > 0 {
		tries = c.pollTries
	}
	return wait, tries
}

// TopUpWalletBLIK is the captured POST /payment wallet top-up, then GET /payment/{id} until it leaves PENDING.
func (c *Client) TopUpWalletBLIK(ctx context.Context, amountPLN, authCode string) error {
	uid, _ := c.dashIDs()
	if uid == "" {
		return fmt.Errorf("client: no user id")
	}
	amount, err := parsePositiveAmount(amountPLN)
	if err != nil {
		return err
	}
	code := digitsOnly(authCode)
	if len(code) != 6 {
		return fmt.Errorf("client: BLIK code must be 6 digits")
	}
	c.report("payments", "Starting BLIK top-up")
	payload := map[string]any{
		"type": "PAYMENT",
		"channel": map[string]any{
			"id":   "APP",
			"name": "APP",
		},
		"beneficiary": map[string]any{
			"id":           uid,
			"referredType": "DIGITAL_WALLET",
		},
		"payer": map[string]any{
			"id": uid,
		},
		"paymentItem": []map[string]any{
			{
				"productOfferingId": "TopUpDigitalWallet",
				"amount": map[string]any{
					"amount":   amount,
					"currency": "pln",
				},
			},
		},
		"totalAmount": map[string]any{
			"amount":   amount,
			"currency": "pln",
		},
		"orderId": newUUID(),
		"paymentMethod": map[string]any{
			"action":     "ONE_CLICK",
			"methodType": "BLIK",
			"authCode":   code,
		},
	}
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := c.postJSONWithHeaders(ctx, c.publicHost+"/payment", true, payload, &created, map[string]string{"version": "2"}); err != nil {
		return err
	}
	if created.ID == "" {
		return fmt.Errorf("client: payment created without id")
	}
	status := strings.ToUpper(strings.TrimSpace(created.Status))
	if status == "" {
		status = "PENDING"
	}
	if err := paymentTerminal(status); err != nil {
		return err
	}
	if status == "COMPLETED" {
		return nil
	}
	c.report("payments", "Waiting for BLIK confirmation")
	wait, tries := c.pollSettings()
	etag := ""
	for i := 0; i < tries; i++ {
		if err := sleepCtx(ctx, wait); err != nil {
			return err
		}
		next, nextETag, err := c.getPaymentStatus(ctx, created.ID, etag)
		if err != nil {
			return err
		}
		if nextETag != "" {
			etag = nextETag
		}
		if next == "" {
			continue
		}
		status = strings.ToUpper(next)
		if err := paymentTerminal(status); err != nil {
			return err
		}
		if status == "COMPLETED" {
			return nil
		}
	}
	return fmt.Errorf("client: BLIK top-up still %s", strings.ToLower(status))
}

func paymentTerminal(status string) error {
	switch status {
	case "PENDING", "PROCESSING", "IN_PROGRESS", "CREATED":
		return nil
	case "COMPLETED":
		return nil
	default:
		return fmt.Errorf("client: BLIK top-up %s", strings.ToLower(status))
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *Client) getPaymentStatus(ctx context.Context, paymentID, etag string) (status, newETag string, err error) {
	raw := c.publicHost + "/payment/" + url.PathEscape(paymentID)
	req, err := c.newRequest(ctx, http.MethodGet, raw, nil, "", true)
	if err != nil {
		return "", "", err
	}
	req.Header.Del("version")
	req.Header.Set("x-api-key", APIKey)
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := c.do(req)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if err := c.refreshTokens(ctx); err != nil {
			return "", "", err
		}
		req, err = c.newRequest(ctx, http.MethodGet, raw, nil, "", true)
		if err != nil {
			return "", "", err
		}
		req.Header.Del("version")
		req.Header.Set("x-api-key", APIKey)
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		resp, err = c.do(req)
		if err != nil {
			return "", "", err
		}
	}
	newETag = resp.Header.Get("ETag")
	if resp.StatusCode == http.StatusNotModified {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return "", newETag, nil
	}
	var pay struct {
		Status string `json:"status"`
	}
	if err := c.readJSON(resp, &pay); err != nil {
		return "", "", err
	}
	return pay.Status, newETag, nil
}

// WithdrawDataSafe is POST /products/{id}/balances/transfer with type BANK_WITHDRAWAL.
// Captured 201, empty body; value is a GB string like GB_TRANSFER.
func (c *Client) WithdrawDataSafe(ctx context.Context, amountGB string) error {
	_, pid := c.dashIDs()
	if pid == "" {
		return fmt.Errorf("client: no product id")
	}
	amount, err := parsePositiveAmount(amountGB)
	if err != nil {
		return err
	}
	if d := c.lastDashboard(); d != nil && d.DataSafe.Amount > 0 && amount > d.DataSafe.Amount+0.0001 {
		return fmt.Errorf("client: you do not have that much data in the Safe")
	}
	value := strings.TrimSpace(strings.ReplaceAll(amountGB, ",", "."))
	if amount == float64(int64(amount)) {
		value = strconv.FormatInt(int64(amount), 10)
	}
	c.report("bank", "Withdrawing data")
	payload := map[string]any{
		"type":  "BANK_WITHDRAWAL",
		"value": value,
	}
	return c.postJSON(ctx, c.publicHost+"/products/"+url.PathEscape(pid)+"/balances/transfer", true, payload, nil)
}
