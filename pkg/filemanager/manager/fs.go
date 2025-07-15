package manager

import (
	"context"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cluster"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/cos"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/ks3"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/local"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/obs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/onedrive"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/oss"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/qiniu"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/remote"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/s3"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver/upyun"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
)

func (m *manager) LocalDriver(policy *ent.StoragePolicy) driver.Handler {
	if policy == nil {
		policy = &ent.StoragePolicy{Type: types.PolicyTypeLocal, Settings: &types.PolicySetting{}}
	}
	return local.New(policy, m.l, m.config)
}

func (m *manager) CastStoragePolicyOnSlave(ctx context.Context, policy *ent.StoragePolicy) *ent.StoragePolicy {
	if !m.stateless {
		return policy
	}

	nodeId := cluster.NodeIdFromContext(ctx)
	if policy.Type == types.PolicyTypeRemote {
		if nodeId != policy.NodeID {
			return policy
		}

		policyCopy := *policy
		policyCopy.Type = types.PolicyTypeLocal
		return &policyCopy
	} else if policy.Type == types.PolicyTypeLocal {
		policyCopy := *policy
		policyCopy.NodeID = nodeId
		policyCopy.Type = types.PolicyTypeRemote
		policyCopy.SetNode(&ent.Node{
			ID:       nodeId,
			Server:   cluster.MasterSiteUrlFromContext(ctx),
			SlaveKey: m.config.Slave().Secret,
		})
		return &policyCopy
	} else if policy.Type == types.PolicyTypeOss {
		policyCopy := *policy
		if policyCopy.Settings != nil {
			policyCopy.Settings.ServerSideEndpoint = ""
		}
	}

	return policy
}

func (m *manager) GetStorageDriver(ctx context.Context, policy *ent.StoragePolicy) (driver.Handler, error) {
	switch policy.Type {
	case types.PolicyTypeLocal:
		return local.New(policy, m.l, m.config), nil
	case types.PolicyTypeRemote:
		return remote.New(ctx, policy, m.settings, m.config, m.l)
	case types.PolicyTypeOss:
		return oss.New(ctx, policy, m.settings, m.config, m.l, m.dep.MimeDetector(ctx))
	case types.PolicyTypeCos:
		return cos.New(ctx, policy, m.settings, m.config, m.l, m.dep.MimeDetector(ctx))
	case types.PolicyTypeS3:
		return s3.New(ctx, policy, m.settings, m.config, m.l, m.dep.MimeDetector(ctx))
	case types.PolicyTypeKs3:
		return ks3.New(ctx, policy, m.settings, m.config, m.l, m.dep.MimeDetector(ctx))
	case types.PolicyTypeObs:
		return obs.New(ctx, policy, m.settings, m.config, m.l, m.dep.MimeDetector(ctx))
	case types.PolicyTypeQiniu:
		return qiniu.New(ctx, policy, m.settings, m.config, m.l, m.dep.MimeDetector(ctx))
	case types.PolicyTypeUpyun:
		return upyun.New(ctx, policy, m.settings, m.config, m.l, m.dep.MimeDetector(ctx))
	case types.PolicyTypeOd:
		return onedrive.New(ctx, policy, m.settings, m.config, m.l, m.dep.CredManager())
	default:
		return nil, ErrUnknownPolicyType
	}
}

func (m *manager) getEntityPolicyDriver(cxt context.Context, e fs.Entity, policyOverwrite *ent.StoragePolicy) (*ent.StoragePolicy, driver.Handler, error) {
	policyID := e.PolicyID()
	var (
		policy *ent.StoragePolicy
		err    error
	)
	if policyID == 0 {
		policy = &ent.StoragePolicy{Type: types.PolicyTypeLocal, Settings: &types.PolicySetting{}}
	} else {
		if policyOverwrite != nil && policyOverwrite.ID == policyID {
			policy = policyOverwrite
		} else {
			policy, err = m.policyClient.GetPolicyByID(cxt, e.PolicyID())
			if err != nil {
				return nil, nil, serializer.NewError(serializer.CodeDBError, "failed to get policy", err)
			}
		}
	}

	d, err := m.GetStorageDriver(cxt, policy)
	if err != nil {
		return nil, nil, err
	}

	return policy, d, nil
}
