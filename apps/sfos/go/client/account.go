package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// User is GET /users.
type User struct {
	ID            string          `json:"id"`
	FirstName     string          `json:"firstName"`
	LastName      string          `json:"lastName"`
	Email         string          `json:"email"`
	Type          string          `json:"type"`
	Enabled       bool            `json:"enabled"`
	PaymentMethod []PaymentMethod `json:"paymentMethod"`
	Person        Person          `json:"person"`
}

// Person is identity on User.
type Person struct {
	Verified  bool   `json:"verified"`
	FullName  string `json:"fullName"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// PaymentMethod is a saved wallet or card.
type PaymentMethod struct {
	ID         string               `json:"id"`
	MethodType string               `json:"methodType"`
	Active     bool                 `json:"active"`
	Details    PaymentMethodDetails `json:"details"`
}

// PaymentMethodDetails holds wallet/card fields.
type PaymentMethodDetails struct {
	Amount           string `json:"amount"`
	MaskedCardNumber string `json:"maskedCardNumber"`
	Issuer           string `json:"issuer"`
	Mode             string `json:"mode"`
}

// Product is one row of GET /products.
type Product struct {
	ID               string           `json:"id"`
	Status           string           `json:"status"`
	OfferingID       string           `json:"offeringId"`
	OfferingName     string           `json:"offeringName"`
	OfferingCategory string           `json:"offeringCategory"`
	SpecificationID  string           `json:"specificationId"`
	AnniversaryDate  string           `json:"anniversaryDate"`
	ActivationDate   string           `json:"activationDate"`
	CurrencyCode     string           `json:"currencyCode"`
	Characteristic   map[string]any   `json:"characteristic"`
	Balances         []ProductBalance `json:"balances"`
}

// ProductBalance is a social / video / music / unlimited meter on a product.
type ProductBalance struct {
	Type          string `json:"type"`
	Name          string `json:"name"`
	Unit          string `json:"unit"`
	CurrentValue  any    `json:"currentValue"`
	StartingValue any    `json:"startingValue"`
}

// Counter is one row of GET /products/{id}/counters.
type Counter struct {
	Type            string  `json:"type"`
	Unit            string  `json:"unit"`
	AnniversaryDate string  `json:"anniversaryDate"`
	CurrentValue    float64 `json:"currentValue"`
	GrantValue      float64 `json:"grantValue"`
	StartingValue   float64 `json:"startingValue"`
}

// DataSafe is GET /products/{id}/bank.
type DataSafe struct {
	Amount         float64          `json:"amount"`
	BalancePeriods []DataSafePeriod `json:"balancePeriods"`
}

// DataSafePeriod is one stash window.
type DataSafePeriod struct {
	ActivationDate string  `json:"activationDate"`
	ExpiryDate     string  `json:"expiryDate"`
	CurrentValue   float64 `json:"currentValue"`
}

// Group is GET /groups.
type Group struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	AdminID string        `json:"adminId"`
	Members []GroupMember `json:"members"`
}

// GroupMember is a family member.
type GroupMember struct {
	ID            string          `json:"id"`
	Alias         string          `json:"alias"`
	Role          string          `json:"role"`
	PrimaryMSISDN string          `json:"primaryMsisdn"`
	Status        string          `json:"status"`
	Products      []MemberProduct `json:"products"`
}

// MemberProduct is the nested product on a group member.
type MemberProduct struct {
	ID              string        `json:"id"`
	Status          string        `json:"status"`
	MSISDN          string        `json:"msisdn"`
	OfferID         string        `json:"offerId"`
	AnniversaryDate string        `json:"anniversaryDate"`
	DataPackage     []DataPackage `json:"dataPackage"`
}

// DataPackage is GB remaining on a group member product.
type DataPackage struct {
	Type         string  `json:"type"`
	CurrentValue float64 `json:"currentValue"`
	GrantValue   float64 `json:"grantValue"`
	Unit         string  `json:"unit"`
}

// SIMList is GET /SIM/list.
type SIMList struct {
	ParentProductID   string    `json:"parentProductId"`
	Offer             string    `json:"offer"`
	MultisimAvailable int       `json:"multisimAvailable"`
	MultisimUsed      int       `json:"multisimUsed"`
	MultisimTotal     int       `json:"multisimTotal"`
	SIMCards          []SIMCard `json:"simCards"`
}

// SIMCard is one SIM.
type SIMCard struct {
	ProductID     string `json:"productId"`
	CardHierarchy string `json:"cardHierarchy"`
	CardType      string `json:"cardType"`
	Label         string `json:"label"`
}

// Dashboard is the logged-in home snapshot.
type Dashboard struct {
	User     User
	Product  Product
	Counters []Counter
	DataSafe DataSafe
	Groups   []Group
	SIMs     SIMList
}

func (p Product) MSISDN() string {
	if p.Characteristic == nil {
		return ""
	}
	if v, ok := p.Characteristic["msisdn"]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

// MainCounter is the plan data ring.
func (d Dashboard) MainCounter() Counter {
	for _, c := range d.Counters {
		if c.Type == "mainCounters" {
			return c
		}
	}
	if len(d.Counters) > 0 {
		return d.Counters[0]
	}
	return Counter{}
}

// RoamingCounter is EU FUP.
func (d Dashboard) RoamingCounter() Counter {
	for _, c := range d.Counters {
		if c.Type == "dataFUPCounters" {
			return c
		}
	}
	return Counter{}
}

// DisplayName prefers person name, then first+last, then alias.
func (d Dashboard) DisplayName() string {
	if d.User.Person.FirstName != "" {
		return d.User.Person.FirstName
	}
	if d.User.FirstName != "" {
		return d.User.FirstName
	}
	if len(d.Groups) > 0 && len(d.Groups[0].Members) > 0 && d.Groups[0].Members[0].Alias != "" {
		return d.Groups[0].Members[0].Alias
	}
	return ""
}

// WalletAmount is Flex funds, if present.
func (d Dashboard) WalletAmount() string {
	for _, pm := range d.User.PaymentMethod {
		if pm.MethodType == "DIGITAL_WALLET" && pm.Details.Amount != "" {
			return pm.Details.Amount
		}
	}
	return ""
}

// CounterGB converts a MB counter to GB (1024).
func CounterGB(c Counter) (left, grant float64) {
	div := 1.0
	if strings.EqualFold(c.Unit, "MB") {
		div = 1024
	}
	return c.CurrentValue / div, c.GrantValue / div
}

// FormatRenewal formats anniversaryDate for the home line.
func FormatRenewal(iso string) string {
	if iso == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, strings.Replace(iso, ".000Z", "Z", 1))
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000Z", iso)
	}
	if err != nil {
		return iso
	}
	return t.Local().Format("02.01.2006 15:04")
}

// FetchUser is GET /users.
func (c *Client) FetchUser(ctx context.Context) (*User, error) {
	var u User
	if err := c.getJSON(ctx, c.publicHost+"/users", true, &u); err != nil {
		return nil, err
	}
	if u.ID != "" {
		c.mu.Lock()
		c.session.UserID = u.ID
		c.mu.Unlock()
		_ = c.saveSession()
	}
	return &u, nil
}

// FetchProducts is GET /products.
func (c *Client) FetchProducts(ctx context.Context) ([]Product, error) {
	var out []Product
	if err := c.getJSON(ctx, c.publicHost+"/products", true, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FetchCounters is GET /products/{id}/counters.
func (c *Client) FetchCounters(ctx context.Context, productID, userID string) ([]Counter, error) {
	raw := c.publicHost + "/products/" + url.PathEscape(productID) + "/counters"
	if userID != "" {
		raw += "?userId=" + url.QueryEscape(userID)
	}
	var out []Counter
	if err := c.getJSON(ctx, raw, true, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FetchDataSafe is GET /products/{id}/bank.
func (c *Client) FetchDataSafe(ctx context.Context, productID string) (*DataSafe, error) {
	var out DataSafe
	if err := c.getJSON(ctx, c.publicHost+"/products/"+url.PathEscape(productID)+"/bank", true, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// FetchGroups is GET /groups.
func (c *Client) FetchGroups(ctx context.Context) ([]Group, error) {
	raw := c.publicHost + "/groups?withProducts=true&withOrchestrators=true"
	var out []Group
	if err := c.getJSON(ctx, raw, true, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FetchSIMs is GET /SIM/list.
func (c *Client) FetchSIMs(ctx context.Context, userID string) (*SIMList, error) {
	raw := c.publicHost + "/SIM/list"
	if userID != "" {
		raw += "?userId=" + url.QueryEscape(userID)
	}
	var out SIMList
	if err := c.getJSON(ctx, raw, true, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// LoadDashboard fetches the logged-in home payload.
func (c *Client) LoadDashboard(ctx context.Context) (*Dashboard, error) {
	c.report("account", "Loading account")
	u, err := c.FetchUser(ctx)
	if err != nil {
		return nil, err
	}
	d := &Dashboard{User: *u}
	c.report("account", "Loading plan")
	products, err := c.FetchProducts(ctx)
	if err != nil {
		return nil, err
	}
	d.Product = pickMobileProduct(products)
	if d.Product.ID != "" {
		counters, err := c.FetchCounters(ctx, d.Product.ID, u.ID)
		if err != nil {
			return nil, err
		}
		d.Counters = counters
		if safe, err := c.FetchDataSafe(ctx, d.Product.ID); err == nil && safe != nil {
			d.DataSafe = *safe
		}
	}
	if groups, err := c.FetchGroups(ctx); err == nil {
		d.Groups = groups
	}
	if sims, err := c.FetchSIMs(ctx, u.ID); err == nil && sims != nil {
		d.SIMs = *sims
	}
	c.report("ready", "Ready")
	c.mu.Lock()
	c.lastDash = d
	c.mu.Unlock()
	return d, nil
}

func pickMobileProduct(products []Product) Product {
	for _, p := range products {
		if strings.EqualFold(p.SpecificationID, "MOBILE_ACCESS") && strings.EqualFold(p.Status, "Active") {
			return p
		}
	}
	for _, p := range products {
		if strings.EqualFold(p.Status, "Active") {
			return p
		}
	}
	if len(products) > 0 {
		return products[0]
	}
	return Product{}
}

// FormatGB renders a GB amount like the Android home ring.
func FormatGB(n float64) string {
	if n == float64(int64(n)) {
		return strconv.FormatInt(int64(n), 10)
	}
	return strconv.FormatFloat(n, 'f', 2, 64)
}

// MustJSON is a test helper.
func MustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
