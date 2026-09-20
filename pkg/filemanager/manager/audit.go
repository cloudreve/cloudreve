package manager

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
)

// auditTimeout bounds a single audit request so a stuck endpoint cannot hang
// share creation.
const auditTimeout = 30 * time.Second

type auditResponse struct {
	Flagged bool   `json:"flagged"`
	Reason  string `json:"reason"`
}

// auditShareEntities submits every auditable entity covered by a share to the
// owning policy's audit endpoint (PolicySetting.AuditEndpoint, 鉴黄). Only
// image entities within AuditMaxSize are sent; a flagged result aborts share
// creation. Endpoint failures are fail-closed — an unavailable audit service
// must not silently bypass the check.
func (l *manager) auditShareEntities(ctx context.Context, files []fs.File) error {
	policyClient := l.dep.StoragePolicyClient()
	mime := l.dep.MimeDetector(ctx)

	seen := map[int]struct{}{}
	policies := map[int]*ent.StoragePolicy{}
	for _, file := range files {
		// The extension lives on the file name; entity sources are blob keys.
		fileMime := mime.TypeByName(file.Name())
		if !strings.HasPrefix(fileMime, "image/") {
			continue
		}
		for _, e := range file.Entities() {
			if _, ok := seen[e.ID()]; ok {
				continue
			}
			seen[e.ID()] = struct{}{}

			if e.Type() != types.EntityTypeVersion || e.Size() == 0 {
				continue
			}

			policy, ok := policies[e.PolicyID()]
			if !ok {
				p, err := policyClient.GetPolicyByID(ctx, e.PolicyID())
				if err != nil {
					return serializer.NewError(serializer.CodeContentAuditFailed, "failed to load entity storage policy", err)
				}
				policy = p
				policies[e.PolicyID()] = p
			}

			endpoint := policy.Settings.AuditEndpoint
			if endpoint == "" {
				continue
			}
			if policy.Settings.AuditMaxSize > 0 && e.Size() > policy.Settings.AuditMaxSize {
				continue
			}

			src, err := l.GetEntitySource(ctx, e.ID(), fs.WithEntity(e))
			if err != nil {
				return serializer.NewError(serializer.CodeContentAuditFailed, "failed to open entity for audit", err)
			}
			flagged, reason, err := callAuditEndpoint(l.dep.RequestClient(request.WithContext(ctx), request.WithTimeout(auditTimeout)),
				endpoint, file.Name(), e.Size(), fileMime, src)
			src.Close()
			if err != nil {
				return err
			}
			if flagged {
				activity.Record(ctx, l.settings, l.dep.ActivityClient(), types.EventContentAuditBlocked,
					activity.File(file.ID()), activity.Extra(map[string]any{
						"entity_id": e.ID(),
						"reason":    reason,
					}))
				return serializer.NewError(serializer.CodeContentAuditRejected, "content rejected by audit", nil)
			}
		}
	}
	return nil
}

// callAuditEndpoint POSTs the entity body to the audit endpoint and returns the
// flagged verdict. Endpoint must be plain http(s); the endpoint answers
// {"flagged": bool, "reason": string}.
func callAuditEndpoint(client request.Client, endpoint, name string, size int64, mimeType string, body io.Reader) (bool, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false, "", serializer.NewError(serializer.CodeContentAuditFailed, "invalid audit endpoint", nil)
	}

	resp := client.Request(http.MethodPost, endpoint, body,
		request.WithContentLength(size),
		request.WithHeader(http.Header{
			"Content-Type":  {mimeType},
			"X-Entity-Name": {name},
			"X-Entity-Size": {strconv.FormatInt(size, 10)},
		}),
	)
	raw, err := resp.CheckHTTPResponse(http.StatusOK).GetResponse()
	if err != nil {
		return false, "", serializer.NewError(serializer.CodeContentAuditFailed, "audit endpoint error", err)
	}

	var result auditResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return false, "", serializer.NewError(serializer.CodeContentAuditFailed, "invalid audit response", err)
	}
	return result.Flagged, result.Reason, nil
}
