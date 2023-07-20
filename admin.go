package hitron

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func (c *CableModem) CMReboot(ctx context.Context) (*Error, error) {
	csrf, err := c.UsersCSRF(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve CSRF token: %w", err)
	}

	o := Error{}

	err = c.sendRequest(ctx, http.MethodPost, "/CM/Reboot",
		url.Values{
			"model": []string{`{"reboot":1}`},
			"csrf":  []string{csrf.CSRF},
		}, &o)
	if err != nil {
		return nil, err
	}

	return &o, nil
}

func (c *CableModem) CMReset(ctx context.Context) (*Error, error) {
	csrf, err := c.UsersCSRF(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve CSRF token: %w", err)
	}

	o := Error{}

	err = c.sendRequest(ctx, http.MethodPost, "/CM/Reboot",
		url.Values{
			"model": []string{`{"factoryReset":1}`},
			"csrf":  []string{csrf.CSRF},
		}, &o)
	if err != nil {
		return nil, err
	}

	return &o, nil
}

func (c *CableModem) CMClearLog(ctx context.Context) (*Error, error) {
	csrf, err := c.UsersCSRF(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve CSRF token: %w", err)
	}

	o := Error{}

	err = c.sendRequest(ctx, http.MethodPost, "/CM/Log",
		url.Values{
			"model":   []string{"[]"},
			"csrf":    []string{csrf.CSRF},
			"_method": []string{"PUT"},
		}, &o)
	if err != nil {
		return nil, err
	}

	return &o, nil
}

// SelfInstall initializes the modem by completing the Easy Connect setup wizard
func (c *CableModem) SelfInstall(ctx context.Context) (*Error, error) {
	o := Error{}

	err := c.sendRequest(ctx, http.MethodPost, "/SelfInstall",
		url.Values{
			"model": []string{fmt.Sprintf(`{"name":"%s","password":"%s"}`, c.credentials.Username, c.credentials.Password)},
		}, &o)
	if err != nil {
		return nil, err
	}

	return &o, nil
}
