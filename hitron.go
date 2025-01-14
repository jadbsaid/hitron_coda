package hitron

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

//go:generate gomplate -c .=apilist.yaml -f methods.go.tmpl -o methods.go

// CableModem represents the Hitron CODA Cable Modem/Router
type CableModem struct {
	base        *url.URL
	hc          *http.Client
	credentials credentials
}

// debugTransport - logs the request and response if debug is enabled
type debugTransport struct {
	rt http.RoundTripper
}

func (t *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	debugLogger := debugLoggerFromContext(req.Context())
	if debugLogger == nil {
		return t.rt.RoundTrip(req)
	}

	drq, _ := httputil.DumpRequest(req, true)
	debugLogger.Logf("request: %s", drq)

	resp, err := t.rt.RoundTrip(req)
	if err == nil {
		drs, _ := httputil.DumpResponse(resp, true)
		debugLogger.Logf("response: %s", drs)
	}

	return resp, err
}

// New instantiates a default CableModem struct
func New(host, username, password string) (*CableModem, error) {
	u, err := url.Parse(fmt.Sprintf("http://%s/1/Device/", host))
	if err != nil {
		return nil, err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Jar: jar,
		// Ignore redirects
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &debugTransport{http.DefaultTransport},
	}

	creds := credentials{username, password}

	return &CableModem{
		credentials: creds,
		base:        u,
		hc:          client,
	}, nil
}

func (c *CableModem) url(s string) *url.URL {
	if len(s) == 0 || c.base == nil {
		return c.base
	}

	if s[0] == '/' {
		s = s[1:]
	}

	p, err := url.Parse(s)
	if err != nil {
		panic(err)
	}

	return c.base.ResolveReference(p)
}

type debugLogger interface {
	Logf(format string, args ...interface{})
}

type debugLoggerKey struct{}

// ContextWithDebugLogger - add a logger for debugging the client
func ContextWithDebugLogger(ctx context.Context,
	l interface {
		Logf(format string, args ...interface{})
	},
) context.Context {
	return context.WithValue(ctx, debugLoggerKey{}, l)
}

type debugLoggerFunc func(format string, args ...interface{})

func (f debugLoggerFunc) Logf(format string, args ...interface{}) {
	f(format, args...)
}

func debugLoggerFromContext(ctx context.Context) debugLogger {
	if l := ctx.Value(debugLoggerKey{}); l != nil {
		dl, ok := l.(debugLogger)
		if ok {
			return dl
		}
	}

	return debugLoggerFunc(func(f string, args ...interface{}) {})
}

func (c *CableModem) getJSON(ctx context.Context, path string, o interface{}) error {
	return c.sendRequest(ctx, http.MethodGet, path, http.NoBody, o)
}

func (c *CableModem) PostJSON(ctx context.Context, path string, body, o interface{}) error {
	return c.sendRequest(ctx, http.MethodPost, path, body, o)
}

func (c *CableModem) sendRequest(ctx context.Context, method, path string, body, o interface{}) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	u := c.url(path).String()

	contentType := ""

	var reqBody io.Reader
	switch b := body.(type) {
	case io.Reader:
		reqBody = b
	case url.Values:
		contentType = "application/x-www-form-urlencoded"
		reqBody = strings.NewReader(b.Encode())
	default:
		return fmt.Errorf("unsupported body type %T", body)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed with status %d: %s (Header: %v)", resp.StatusCode, string(b), resp.Header)
	}

	err = json.Unmarshal(b, o)
	if err != nil {
		return fmt.Errorf("JSON decoding failed: %w", err)
	}

	return nil
}

func atoi64(s string) int64 {
	i, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)

	return i
}

func atof64(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)

	return f
}

//nolint:gomnd
const (
	_byte = 1 << (10 * iota)
	kib
	mib
	gib
	tib
	pib
	eib
)

func formattedBytesToInt64(s string) int64 {
	i, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err == nil {
		return i
	}

	s = strings.TrimSuffix(s, " Bytes")
	if len(s) <= 1 {
		return atoi64(s)
	}

	switch s[len(s)-1] {
	case 'B':
		i = int64(atof64(s[:len(s)-1]))
	case 'K':
		i = int64(atof64(s[:len(s)-1]) * kib)
	case 'M':
		i = int64(atof64(s[:len(s)-1]) * mib)
	case 'G':
		i = int64(atof64(s[:len(s)-1]) * gib)
	case 'T':
		i = int64(atof64(s[:len(s)-1]) * tib)
	case 'P':
		i = int64(atof64(s[:len(s)-1]) * pib)
	case 'E':
		i = int64(atof64(s[:len(s)-1]) * eib)
	default:
		i = int64(atof64(s))
	}

	return i
}

func byteSize(bytes uint64) string {
	unit := ""
	value := float64(bytes)

	switch {
	case bytes >= eib:
		unit = "E"
		value /= eib
	case bytes >= pib:
		unit = "P"
		value /= pib
	case bytes >= tib:
		unit = "T"
		value /= tib
	case bytes >= gib:
		unit = "G"
		value /= gib
	case bytes >= mib:
		unit = "M"
		value /= mib
	case bytes >= kib:
		unit = "K"
		value /= kib
	case bytes >= _byte:
		unit = "B"
	case bytes == 0:
		return "0B"
	}

	result := strconv.FormatFloat(value, 'f', 1, 64)
	result = strings.TrimSuffix(result, ".0")

	return result + unit
}
