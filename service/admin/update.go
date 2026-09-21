package admin

import (
	"context"
	"time"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/cloudreve/Cloudreve/v4/pkg/updatecheck"
	"github.com/gin-gonic/gin"
)

type (
	// UpdateCheckService returns the latest GitHub release for this server.
	UpdateCheckService  struct{}
	UpdateCheckParamCtx struct{}

	// UpdateApplyService performs an in-place self-update and restarts.
	UpdateApplyService  struct{}
	UpdateApplyParamCtx struct{}

	// UpdateCheckResult combines release info with self-update capability.
	UpdateCheckResult struct {
		*updatecheck.ReleaseInfo
		SelfUpdate bool   `json:"self_update"`
		Container  bool   `json:"container"`
		Reason     string `json:"reason,omitempty"`
	}
)

// Check compares the running version against the newest GitHub release.
func (service *UpdateCheckService) Check(c *gin.Context) (*UpdateCheckResult, error) {
	dep := dependency.FromContext(c)
	rel, err := updatecheck.LatestRelease(c, dep.KV())
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Failed to check for updates: "+err.Error(), err)
	}
	ok, reason := updatecheck.SelfUpdateSupported()
	return &UpdateCheckResult{
		ReleaseInfo: rel,
		SelfUpdate:  ok,
		Container:   updatecheck.InContainer(),
		Reason:      reason,
	}, nil
}

// Apply downloads, verifies and installs the latest release, then restarts the
// process. The response is returned before the binary swap starts so the admin
// sees a confirmation instead of a dropped connection.
func (service *UpdateApplyService) Apply(c *gin.Context) (*updatecheck.ReleaseInfo, error) {
	if ok, reason := updatecheck.SelfUpdateSupported(); !ok {
		return nil, serializer.NewError(serializer.CodeParamErr, "Self-update is not supported in this environment: "+reason, nil)
	}

	dep := dependency.FromContext(c)
	rel, err := updatecheck.LatestRelease(c, dep.KV())
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "Failed to check for updates: "+err.Error(), err)
	}
	if !rel.Newer {
		return nil, serializer.NewError(serializer.CodeParamErr, "Already on the latest version", nil)
	}

	dep.Logger().Info("Applying server update to v%s", rel.Version)
	logger := dep.Logger()
	go func() {
		// Give the HTTP response a moment to flush before the process exits.
		time.Sleep(800 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		if err := updatecheck.ApplyUpdate(ctx, rel); err != nil {
			logger.Error("Self-update failed: %s", err)
		}
	}()

	return rel, nil
}
