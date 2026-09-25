package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef void (*orangeflex_progress_fn)(const char *step, const char *message, int64_t bytes, int64_t total);

static void orangeflex_call_progress(orangeflex_progress_fn fn, const char *step, const char *message, int64_t bytes, int64_t total) {
	if (fn) {
		fn(step, message, bytes, total);
	}
}
*/
import "C"

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"x.x/x/orangeflex/client"
)

func main() {}

var (
	mu         sync.Mutex
	cacheDir   string
	flex       *client.Client
	dash       *client.Dashboard
	lastErr    string
	ready      bool
	progressFn C.orangeflex_progress_fn
	otpTimeout int
	listJSON   string
	listName   string
	extraJSON  string
)

type memberSnap struct {
	alias   string
	role    string
	msisdn  string
	status  string
	leftGB  string
	grantGB string
}

func cstr(s string) *C.char {
	return C.CString(s)
}

func gostr(p *C.char) string {
	if p == nil {
		return ""
	}
	return C.GoString(p)
}

func setErr(err error) C.int {
	mu.Lock()
	defer mu.Unlock()
	if err == nil {
		lastErr = ""
		return 0
	}
	lastErr = err.Error()
	return -1
}

func emitProgress(ev client.ProgressEvent) {
	mu.Lock()
	fn := progressFn
	mu.Unlock()
	if fn == nil {
		return
	}
	step := C.CString(ev.Step)
	msg := C.CString(ev.Message)
	C.orangeflex_call_progress(fn, step, msg, 0, 0)
	C.free(unsafe.Pointer(step))
	C.free(unsafe.Pointer(msg))
}

func networkFlagPath() string {
	if cacheDir == "" {
		return ""
	}
	return filepath.Join(cacheDir, "network.allowed")
}

func persistNetwork(allow bool) {
	p := networkFlagPath()
	if p == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p), 0750)
	val := "0\n"
	if allow {
		val = "1\n"
	}
	_ = os.WriteFile(p, []byte(val), 0640)
}

func loadNetworkFlag() bool {
	b, err := os.ReadFile(networkFlagPath())
	return err == nil && strings.TrimSpace(string(b)) == "1"
}

func networkPromptNeeded() bool {
	p := networkFlagPath()
	if p == "" {
		return true
	}
	_, err := os.Stat(p)
	return err != nil
}

func snapshotDashboard(d *client.Dashboard) {
	mu.Lock()
	dash = d
	mu.Unlock()
}

func snapshotExtras(c *client.Client) {
	if c == nil {
		mu.Lock()
		extraJSON = "{}"
		mu.Unlock()
		return
	}
	b, err := c.ExtraJSON()
	mu.Lock()
	if err != nil || len(b) == 0 {
		extraJSON = "{}"
	} else {
		extraJSON = string(b)
	}
	mu.Unlock()
}

func renewalParts(iso string) (date, time string) {
	s := client.FormatRenewal(iso)
	if s == "" {
		return "", ""
	}
	i := strings.LastIndex(s, " ")
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+1:]
}

func memberPackage(m client.GroupMember) (left, grant string) {
	for _, p := range m.Products {
		for _, pkg := range p.DataPackage {
			if pkg.Type == "plData" || pkg.Type == "mainCounters" {
				return client.FormatGB(pkg.CurrentValue), client.FormatGB(pkg.GrantValue)
			}
		}
		if len(p.DataPackage) > 0 {
			pkg := p.DataPackage[0]
			return client.FormatGB(pkg.CurrentValue), client.FormatGB(pkg.GrantValue)
		}
	}
	return "", ""
}

func membersFrom(d *client.Dashboard) []memberSnap {
	if d == nil || len(d.Groups) == 0 {
		return nil
	}
	out := make([]memberSnap, 0, len(d.Groups[0].Members))
	for _, m := range d.Groups[0].Members {
		left, grant := memberPackage(m)
		out = append(out, memberSnap{
			alias:   m.Alias,
			role:    m.Role,
			msisdn:  m.PrimaryMSISDN,
			status:  m.Status,
			leftGB:  left,
			grantGB: grant,
		})
	}
	return out
}

//export OrangeFlex_SetProgressHandler
func OrangeFlex_SetProgressHandler(fn C.orangeflex_progress_fn) {
	mu.Lock()
	progressFn = fn
	mu.Unlock()
}

//export OrangeFlex_SetCacheDir
func OrangeFlex_SetCacheDir(dir *C.char) {
	mu.Lock()
	cacheDir = gostr(dir)
	mu.Unlock()
	if loadNetworkFlag() {
		client.SetNetworkAllowed(true)
	}
}

//export OrangeFlex_SetNetworkAllowed
func OrangeFlex_SetNetworkAllowed(allow C.int) {
	client.SetNetworkAllowed(allow != 0)
	persistNetwork(allow != 0)
	mu.Lock()
	c := flex
	mu.Unlock()
	if c != nil {
		c.PersistNetwork(allow != 0)
	}
}

//export OrangeFlex_NetworkAllowed
func OrangeFlex_NetworkAllowed() C.int {
	if client.NetworkAllowed() {
		return 1
	}
	return 0
}

//export OrangeFlex_NetworkPromptNeeded
func OrangeFlex_NetworkPromptNeeded() C.int {
	if networkPromptNeeded() {
		return 1
	}
	return 0
}

//export OrangeFlex_Init
func OrangeFlex_Init() C.int {
	if !client.NetworkAllowed() {
		mu.Lock()
		lastErr = "network permission not granted"
		ready = false
		mu.Unlock()
		return -1
	}

	mu.Lock()
	dir := cacheDir
	mu.Unlock()

	opts := []client.Option{
		client.WithProgress(emitProgress),
	}
	if dir != "" {
		opts = append(opts, client.WithCacheDir(dir))
	}

	c, err := client.New(opts...)
	if err != nil {
		mu.Lock()
		lastErr = err.Error()
		ready = false
		flex = nil
		dash = nil
		mu.Unlock()
		return -1
	}

	mu.Lock()
	flex = c
	lastErr = ""
	ready = true
	mu.Unlock()

	if c.LoggedIn() {
		d, err := c.LoadDashboard(context.Background())
		if err != nil {
			mu.Lock()
			lastErr = err.Error()
			mu.Unlock()
			return 0
		}
		snapshotDashboard(d)
		snapshotExtras(c)
	}
	return 0
}

//export OrangeFlex_Ready
func OrangeFlex_Ready() C.int {
	mu.Lock()
	defer mu.Unlock()
	if ready {
		return 1
	}
	return 0
}

//export OrangeFlex_LoggedIn
func OrangeFlex_LoggedIn() C.int {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c != nil && c.LoggedIn() {
		return 1
	}
	return 0
}

//export OrangeFlex_LastError
func OrangeFlex_LastError() *C.char {
	mu.Lock()
	err := lastErr
	mu.Unlock()
	return cstr(err)
}

//export OrangeFlex_Free
func OrangeFlex_Free(p *C.char) {
	if p != nil {
		C.free(unsafe.Pointer(p))
	}
}

//export OrangeFlex_LoginStart
func OrangeFlex_LoginStart(to *C.char) C.int {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c == nil {
		return setErr(errNotReady())
	}
	st, err := c.LoginStart(context.Background(), gostr(to))
	if err != nil {
		return setErr(err)
	}
	mu.Lock()
	otpTimeout = st.Timeout
	lastErr = ""
	mu.Unlock()
	return 0
}

type notReadyError struct{}

func (notReadyError) Error() string { return "client not initialized" }

func errNotReady() error { return notReadyError{} }

//export OrangeFlex_LoginSubmitOTP
func OrangeFlex_LoginSubmitOTP(otp *C.char) C.int {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c == nil {
		return setErr(errNotReady())
	}
	res, err := c.LoginSubmitOTP(context.Background(), gostr(otp))
	if err != nil {
		return setErr(err)
	}
	mu.Lock()
	otpTimeout = res.Timeout
	lastErr = ""
	mu.Unlock()
	if res.NeedNextOTP {
		return 1
	}
	if res.RestartSMS {
		return 2
	}
	if !res.LoggedIn {
		mu.Lock()
		lastErr = "login incomplete"
		mu.Unlock()
		return -1
	}
	d, err := c.LoadDashboard(context.Background())
	if err != nil {
		mu.Lock()
		lastErr = err.Error()
		mu.Unlock()
		return 0
	}
	snapshotDashboard(d)
	snapshotExtras(c)
	return 0
}

//export OrangeFlex_LoginTimeout
func OrangeFlex_LoginTimeout() C.int {
	mu.Lock()
	defer mu.Unlock()
	return C.int(otpTimeout)
}

//export OrangeFlex_PendingEmailMask
func OrangeFlex_PendingEmailMask() *C.char {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c == nil {
		return cstr("")
	}
	return cstr(c.PendingEmailMask())
}

//export OrangeFlex_PendingMSISDN
func OrangeFlex_PendingMSISDN() *C.char {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c == nil {
		return cstr("")
	}
	return cstr(c.PendingMSISDN())
}

func reloadAccount(c *client.Client) error {
	d, err := c.LoadDashboard(context.Background())
	if err != nil {
		return err
	}
	snapshotDashboard(d)
	snapshotExtras(c)
	return nil
}

//export OrangeFlex_Refresh
func OrangeFlex_Refresh() C.int {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c == nil || !c.LoggedIn() {
		return setErr(errNotReady())
	}
	if err := reloadAccount(c); err != nil {
		return setErr(err)
	}
	return setErr(nil)
}

//export OrangeFlex_Logout
func OrangeFlex_Logout() C.int {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c == nil {
		return 0
	}
	if err := c.Logout(); err != nil {
		return setErr(err)
	}
	mu.Lock()
	dash = nil
	extraJSON = "{}"
	listJSON = "[]"
	listName = ""
	lastErr = ""
	mu.Unlock()
	return 0
}

//export OrangeFlex_WipeData
func OrangeFlex_WipeData() C.int {
	client.SetNetworkAllowed(false)
	mu.Lock()
	c := flex
	dir := cacheDir
	flex = nil
	dash = nil
	ready = false
	extraJSON = "{}"
	listJSON = "[]"
	listName = ""
	mu.Unlock()
	if c != nil {
		if err := c.WipeData(); err != nil {
			return setErr(err)
		}
	} else if dir != "" {
		_ = os.RemoveAll(dir)
	}
	mu.Lock()
	lastErr = ""
	mu.Unlock()
	return 0
}

//export OrangeFlex_SetLanguage
func OrangeFlex_SetLanguage(lang *C.char) {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c != nil {
		c.SetLanguage(gostr(lang))
	}
}

//export OrangeFlex_Language
func OrangeFlex_Language() *C.char {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c == nil {
		return cstr("en")
	}
	return cstr(c.Language())
}

func dashString(fn func(*client.Dashboard) string) *C.char {
	mu.Lock()
	d := dash
	mu.Unlock()
	if d == nil {
		return cstr("")
	}
	return cstr(fn(d))
}

//export OrangeFlex_DisplayName
func OrangeFlex_DisplayName() *C.char {
	return dashString(func(d *client.Dashboard) string { return d.DisplayName() })
}

//export OrangeFlex_FirstName
func OrangeFlex_FirstName() *C.char {
	return dashString(func(d *client.Dashboard) string { return d.User.FirstName })
}

//export OrangeFlex_LastName
func OrangeFlex_LastName() *C.char {
	return dashString(func(d *client.Dashboard) string { return d.User.LastName })
}

//export OrangeFlex_Email
func OrangeFlex_Email() *C.char {
	return dashString(func(d *client.Dashboard) string { return d.User.Email })
}

//export OrangeFlex_MSISDN
func OrangeFlex_MSISDN() *C.char {
	return dashString(func(d *client.Dashboard) string { return d.Product.MSISDN() })
}

//export OrangeFlex_OfferingName
func OrangeFlex_OfferingName() *C.char {
	return dashString(func(d *client.Dashboard) string { return d.Product.OfferingName })
}

//export OrangeFlex_ProductStatus
func OrangeFlex_ProductStatus() *C.char {
	return dashString(func(d *client.Dashboard) string { return d.Product.Status })
}

//export OrangeFlex_RenewalDate
func OrangeFlex_RenewalDate() *C.char {
	return dashString(func(d *client.Dashboard) string {
		date, _ := renewalParts(d.Product.AnniversaryDate)
		if date == "" {
			date, _ = renewalParts(d.MainCounter().AnniversaryDate)
		}
		return date
	})
}

//export OrangeFlex_RenewalTime
func OrangeFlex_RenewalTime() *C.char {
	return dashString(func(d *client.Dashboard) string {
		_, tm := renewalParts(d.Product.AnniversaryDate)
		if tm == "" {
			_, tm = renewalParts(d.MainCounter().AnniversaryDate)
		}
		return tm
	})
}

//export OrangeFlex_LeftGB
func OrangeFlex_LeftGB() *C.char {
	return dashString(func(d *client.Dashboard) string {
		left, _ := client.CounterGB(d.MainCounter())
		return client.FormatGB(left)
	})
}

//export OrangeFlex_GrantGB
func OrangeFlex_GrantGB() *C.char {
	return dashString(func(d *client.Dashboard) string {
		_, grant := client.CounterGB(d.MainCounter())
		return client.FormatGB(grant)
	})
}

//export OrangeFlex_WalletAmount
func OrangeFlex_WalletAmount() *C.char {
	return dashString(func(d *client.Dashboard) string { return d.WalletAmount() })
}

//export OrangeFlex_DataSafeGB
func OrangeFlex_DataSafeGB() *C.char {
	return dashString(func(d *client.Dashboard) string {
		return client.FormatGB(d.DataSafe.Amount)
	})
}

//export OrangeFlex_SIMCount
func OrangeFlex_SIMCount() C.int {
	mu.Lock()
	d := dash
	mu.Unlock()
	if d == nil {
		return 0
	}
	return C.int(len(d.SIMs.SIMCards))
}

func simAt(index C.int) (client.SIMCard, bool) {
	mu.Lock()
	d := dash
	mu.Unlock()
	i := int(index)
	if d == nil || i < 0 || i >= len(d.SIMs.SIMCards) {
		return client.SIMCard{}, false
	}
	return d.SIMs.SIMCards[i], true
}

//export OrangeFlex_SIMHierarchy
func OrangeFlex_SIMHierarchy(index C.int) *C.char {
	s, ok := simAt(index)
	if !ok {
		return cstr("")
	}
	return cstr(s.CardHierarchy)
}

//export OrangeFlex_SIMType
func OrangeFlex_SIMType(index C.int) *C.char {
	s, ok := simAt(index)
	if !ok {
		return cstr("")
	}
	return cstr(s.CardType)
}

//export OrangeFlex_SIMLabel
func OrangeFlex_SIMLabel(index C.int) *C.char {
	s, ok := simAt(index)
	if !ok {
		return cstr("")
	}
	return cstr(s.Label)
}

//export OrangeFlex_MultisimUsed
func OrangeFlex_MultisimUsed() C.int {
	mu.Lock()
	d := dash
	mu.Unlock()
	if d == nil {
		return 0
	}
	return C.int(d.SIMs.MultisimUsed)
}

//export OrangeFlex_MultisimTotal
func OrangeFlex_MultisimTotal() C.int {
	mu.Lock()
	d := dash
	mu.Unlock()
	if d == nil {
		return 0
	}
	return C.int(d.SIMs.MultisimTotal)
}

//export OrangeFlex_MultisimAvailable
func OrangeFlex_MultisimAvailable() C.int {
	mu.Lock()
	d := dash
	mu.Unlock()
	if d == nil {
		return 0
	}
	return C.int(d.SIMs.MultisimAvailable)
}

//export OrangeFlex_GroupType
func OrangeFlex_GroupType() *C.char {
	return dashString(func(d *client.Dashboard) string {
		if len(d.Groups) == 0 {
			return ""
		}
		return d.Groups[0].Type
	})
}

func memberAt(index C.int) (memberSnap, bool) {
	mu.Lock()
	d := dash
	mu.Unlock()
	members := membersFrom(d)
	i := int(index)
	if i < 0 || i >= len(members) {
		return memberSnap{}, false
	}
	return members[i], true
}

//export OrangeFlex_MemberCount
func OrangeFlex_MemberCount() C.int {
	mu.Lock()
	d := dash
	mu.Unlock()
	return C.int(len(membersFrom(d)))
}

//export OrangeFlex_MemberAlias
func OrangeFlex_MemberAlias(index C.int) *C.char {
	m, ok := memberAt(index)
	if !ok {
		return cstr("")
	}
	return cstr(m.alias)
}

//export OrangeFlex_MemberRole
func OrangeFlex_MemberRole(index C.int) *C.char {
	m, ok := memberAt(index)
	if !ok {
		return cstr("")
	}
	return cstr(m.role)
}

//export OrangeFlex_MemberMSISDN
func OrangeFlex_MemberMSISDN(index C.int) *C.char {
	m, ok := memberAt(index)
	if !ok {
		return cstr("")
	}
	return cstr(m.msisdn)
}

//export OrangeFlex_MemberStatus
func OrangeFlex_MemberStatus(index C.int) *C.char {
	m, ok := memberAt(index)
	if !ok {
		return cstr("")
	}
	return cstr(m.status)
}

//export OrangeFlex_MemberLeftGB
func OrangeFlex_MemberLeftGB(index C.int) *C.char {
	m, ok := memberAt(index)
	if !ok {
		return cstr("")
	}
	return cstr(m.leftGB)
}

//export OrangeFlex_MemberGrantGB
func OrangeFlex_MemberGrantGB(index C.int) *C.char {
	m, ok := memberAt(index)
	if !ok {
		return cstr("")
	}
	return cstr(m.grantGB)
}

//export OrangeFlex_ExtraJSON
func OrangeFlex_ExtraJSON() *C.char {
	mu.Lock()
	s := extraJSON
	mu.Unlock()
	if s == "" {
		s = "{}"
	}
	return cstr(s)
}

//export OrangeFlex_LoadList
func OrangeFlex_LoadList(name *C.char) C.int {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c == nil {
		return setErr(errNotReady())
	}
	n := gostr(name)
	if err := c.LoadList(context.Background(), n); err != nil {
		return setErr(err)
	}
	items := c.ListItems(n)
	b, err := json.Marshal(items)
	if err != nil {
		return setErr(err)
	}
	mu.Lock()
	listJSON = string(b)
	listName = n
	lastErr = ""
	mu.Unlock()
	return 0
}

//export OrangeFlex_ListJSON
func OrangeFlex_ListJSON() *C.char {
	mu.Lock()
	s := listJSON
	mu.Unlock()
	if s == "" {
		s = "[]"
	}
	return cstr(s)
}

//export OrangeFlex_ListName
func OrangeFlex_ListName() *C.char {
	mu.Lock()
	s := listName
	mu.Unlock()
	return cstr(s)
}

//export OrangeFlex_TransferData
func OrangeFlex_TransferData(to, value, message *C.char) C.int {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c == nil {
		return setErr(errNotReady())
	}
	if err := c.TransferData(context.Background(), gostr(to), gostr(value), gostr(message)); err != nil {
		return setErr(err)
	}
	if err := reloadAccount(c); err != nil {
		return setErr(err)
	}
	return setErr(nil)
}

//export OrangeFlex_TopUpBlik
func OrangeFlex_TopUpBlik(amount, authCode *C.char) C.int {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c == nil {
		return setErr(errNotReady())
	}
	if err := c.TopUpWalletBLIK(context.Background(), gostr(amount), gostr(authCode)); err != nil {
		return setErr(err)
	}
	if err := reloadAccount(c); err != nil {
		return setErr(err)
	}
	return setErr(nil)
}

//export OrangeFlex_WithdrawDataSafe
func OrangeFlex_WithdrawDataSafe(amount *C.char) C.int {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c == nil {
		return setErr(errNotReady())
	}
	if err := c.WithdrawDataSafe(context.Background(), gostr(amount)); err != nil {
		return setErr(err)
	}
	if err := reloadAccount(c); err != nil {
		return setErr(err)
	}
	return setErr(nil)
}

//export OrangeFlex_MarkMessagesRead
func OrangeFlex_MarkMessagesRead() C.int {
	mu.Lock()
	c := flex
	mu.Unlock()
	if c == nil {
		return setErr(errNotReady())
	}
	if err := c.MarkMessagesRead(context.Background()); err != nil {
		return setErr(err)
	}
	return setErr(nil)
}
