// Package sms delivers verification codes through a generic HTTP SMS
// gateway. The endpoint URL and request body are admin-configured
// templates accepting `{phone}` and `{code}` placeholders, so any
// provider reachable over plain HTTP can be integrated without a
// vendor SDK.
package sms

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

// SendCode dispatches a verification code through the configured gateway.
// For GET gateways the rendered endpoint is requested with no body; for
// POST the rendered body template is sent with the configured headers.
func SendCode(ctx context.Context, client request.Client, gw *setting.SmsGateway, phone, code string) error {
	if !gw.Enabled || gw.Endpoint == "" {
		return fmt.Errorf("sms gateway not enabled or not configured")
	}

	render := func(tpl string) string {
		return strings.NewReplacer("{phone}", phone, "{code}", code).Replace(tpl)
	}

	endpoint := render(gw.Endpoint)
	// The endpoint is admin-configured, but DNS can still resolve it to a
	// private address — reject before issuing the request.
	if err := request.ValidateExternalURL(ctx, endpoint, request.SSRFOptions{}); err != nil {
		return fmt.Errorf("sms endpoint rejected: %w", err)
	}

	header := http.Header{}
	for _, line := range strings.Split(gw.Headers, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if k = strings.TrimSpace(k); !ok || k == "" {
			continue
		}
		header.Set(k, strings.TrimSpace(v))
	}

	var body *strings.Reader
	method := gw.Method
	if method == http.MethodGet {
		body = strings.NewReader("")
	} else {
		method = http.MethodPost
		body = strings.NewReader(render(gw.BodyTemplate))
	}

	resp, err := client.
		Request(method, endpoint, body,
			request.WithContext(ctx),
			request.WithTimeout(15*time.Second),
			request.WithHeader(header),
			request.WithContentLength(int64(body.Len())),
		).
		CheckHTTPResponse(http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent).
		GetResponse()
	if err != nil {
		return fmt.Errorf("sms gateway request failed: %w", err)
	}
	_ = resp
	return nil
}
