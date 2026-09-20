package dbfs

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/storagepolicy"
	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
)

const MaxFileNameLength = 256

// validateFileName validates the file name. When the policy allows native
// names, characters only forbidden on Windows filesystems are accepted;
// path separators and dot-names stay illegal regardless (#3065).
func validateFileName(name string, policy *ent.StoragePolicy) error {
	if len(name) >= MaxFileNameLength || len(name) == 0 {
		return fmt.Errorf("length of name must be between 1 and 255")
	}

	if strings.ContainsAny(name, "\\/") {
		return fmt.Errorf("name contains path separators")
	}

	nativeAllowed := policy != nil && policy.Settings.AllowNativeName
	if !nativeAllowed && strings.ContainsAny(name, ":*?\"<>|") {
		return fmt.Errorf("name contains illegal characters")
	}

	if name == "." || name == ".." {
		return fmt.Errorf("name cannot be only dot")
	}

	return nil
}

// validateExtension validates the file extension.
func validateExtension(name string, policy *ent.StoragePolicy) error {
	if len(policy.Settings.FileType) == 0 {
		return nil
	}

	inList := util.IsInExtensionList(policy.Settings.FileType, name)
	if (policy.Settings.IsFileTypeDenyList && inList) || (!policy.Settings.IsFileTypeDenyList && !inList) {
		return fmt.Errorf("file extension is not allowed")
	}

	return nil
}

func validateFileNameRegexp(name string, policy *ent.StoragePolicy) error {
	if policy.Settings.NameRegexp == "" {
		return nil
	}

	match, err := regexp.MatchString(policy.Settings.NameRegexp, name)
	if err != nil {
		return fmt.Errorf("invalid file name regexp: %s", err)
	}

	if (policy.Settings.IsNameRegexpDenyList && match) || (!policy.Settings.IsNameRegexpDenyList && !match) {
		return fmt.Errorf("file name is not allowed by regexp")
	}

	return nil
}

// validateFileSize validates the file size.
func validateFileSize(size int64, policy *ent.StoragePolicy) error {
	if policy.MaxSize == 0 {
		return nil
	} else if size > policy.MaxSize {
		return fs.ErrFileSizeTooBig
	}

	return nil
}

// validateNewFile validates the upload request.
func validateNewFile(fileName string, size int64, policy *ent.StoragePolicy) error {
	if err := validateFileName(fileName, policy); err != nil {
		return fs.ErrIllegalObjectName.WithError(err)
	}

	if err := validateExtension(fileName, policy); err != nil {
		return fs.ErrIllegalObjectName.WithError(err)
	}

	if err := validateFileNameRegexp(fileName, policy); err != nil {
		return fs.ErrIllegalObjectName.WithError(err)
	}

	if err := validateFileSize(size, policy); err != nil {
		return err
	}

	return nil
}

func (f *DBFS) validateUserCapacity(ctx context.Context, size int64, u *ent.User) error {
	capacity, err := f.Capacity(ctx, u)
	if err != nil {
		return fmt.Errorf("failed to get user capacity: %s", err)
	}

	return f.validateUserCapacityRaw(ctx, size, capacity)
}

// validateUserCapacityRaw validates the user capacity, but does not fetch the capacity.
func (f *DBFS) validateUserCapacityRaw(ctx context.Context, size int64, capacity *fs.Capacity) error {
	if capacity.Used+size > capacity.Total {
		f.record(ctx, types.EventUserExceedQuotaNotified, activity.Extra(map[string]any{
			"size": size, "capacity": capacity.Total,
		}))
		return fs.ErrInsufficientCapacity
	}
	return nil
}

// validatePolicyCapacity checks that storing `size` more bytes under `policy`
// stays within the policy's MaxTotalSize cap, fetching current usage first.
func (f *DBFS) validatePolicyCapacity(ctx context.Context, size int64, policy *ent.StoragePolicy) error {
	if policy.Settings.MaxTotalSize <= 0 {
		return nil
	}

	_, used, err := f.fileClient.CountEntityByStoragePolicyID(ctx, policy.ID)
	if err != nil {
		return fmt.Errorf("failed to get storage policy usage: %w", err)
	}
	return f.validatePolicyCapacityRaw(size, policy, int64(used))
}

// validatePolicyCapacityRaw validates the policy capacity against a
// caller-supplied usage figure — needed when new entities are being written
// inside a transaction the usage query cannot see yet. The canonical
// ErrInsufficientCapacity is returned unwrapped so upstream errors.Is
// checks still match; the policy detail is logged server-side.
func (f *DBFS) validatePolicyCapacityRaw(size int64, policy *ent.StoragePolicy, used int64) error {
	if policyCap := policy.Settings.MaxTotalSize; policyCap > 0 && used+size > policyCap {
		f.l.Warning("storage policy %q is full (%d + %d > %d)", policy.Name, used, size, policyCap)
		return fs.ErrInsufficientCapacity
	}
	return nil
}

// maxOverflowHops bounds the overflow-chain walk so misconfigured cycles or
// long chains cannot spin the upload path.
const maxOverflowHops = 16

// overflowChain returns the ordered concrete policies an upload may spill
// into: `policy` itself first, then each `overflow_policy_id` hop. Suspended
// members are skipped; load-balance members resolve to a weighted child. The
// walk stops at a missing hop, a cycle, or maxOverflowHops.
func (f *DBFS) overflowChain(ctx context.Context, policy *ent.StoragePolicy) []*ent.StoragePolicy {
	sc, _ := inventory.InheritTx(ctx, f.storagePolicyClient)
	seen := map[int]bool{policy.ID: true}
	chain := make([]*ent.StoragePolicy, 0, 4)
	cur := policy
	for hops := 0; cur != nil && hops < maxOverflowHops; hops++ {
		if cur.Type == types.PolicyTypeLoadBalance {
			if child, err := sc.ResolveLoadBalance(ctx, cur); err == nil &&
				child.Status == storagepolicy.StatusActive && !seen[child.ID] {
				seen[child.ID] = true
				chain = append(chain, child)
			}
		} else if cur.Status == storagepolicy.StatusActive {
			chain = append(chain, cur)
		}

		nextID := cur.Settings.OverflowPolicyID
		if nextID == 0 || seen[nextID] {
			break
		}
		seen[nextID] = true
		next, err := sc.GetPolicyByID(ctx, nextID)
		if err != nil {
			break
		}
		cur = next
	}
	return chain
}

// resolveOverflowPolicy picks the first chain member with headroom for
// `size` more bytes. When every member is full it returns the last one, so
// the caller's own capacity check still reports the canonical error.
func (f *DBFS) resolveOverflowPolicy(ctx context.Context, policy *ent.StoragePolicy, size int64) *ent.StoragePolicy {
	chain := f.overflowChain(ctx, policy)
	for _, p := range chain {
		if err := f.validatePolicyCapacity(ctx, size, p); err == nil {
			if p.ID != policy.ID {
				f.l.Info("storage policy %q full, upload overflows to %q", policy.Name, p.Name)
			}
			return p
		}
	}
	if len(chain) == 0 {
		return policy
	}
	return chain[len(chain)-1]
}

// chainHeadroom returns the aggregate bytes still storable across the
// policy's overflow chain; math.MaxInt64 when any member is uncapped.
func (f *DBFS) chainHeadroom(ctx context.Context, policy *ent.StoragePolicy) (int64, error) {
	total := int64(0)
	for _, p := range f.overflowChain(ctx, policy) {
		if p.Settings.MaxTotalSize <= 0 {
			return math.MaxInt64, nil
		}
		_, used, err := f.fileClient.CountEntityByStoragePolicyID(ctx, p.ID)
		if err != nil {
			return 0, fmt.Errorf("failed to get storage policy usage: %w", err)
		}
		total += max(p.Settings.MaxTotalSize-int64(used), 0)
	}
	return total, nil
}
