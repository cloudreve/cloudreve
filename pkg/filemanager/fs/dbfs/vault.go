package dbfs

import (
	"context"
	"strconv"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/samber/lo"
)

const (
	// MetadataVault marks a root-level folder as the owner's private space.
	// Contents are not flagged individually; membership is decided by the
	// ancestor chain so it stays correct across moves and copies.
	MetadataVault = MetadataSysPrefix + "vault"

	// VaultUnlockCachePrefix prefixes the cache key recording an unlocked
	// private space for a user ID.
	VaultUnlockCachePrefix = "vault_unlocked_"

	// VaultUnlockTTL is the unlock validity in seconds.
	VaultUnlockTTL = 1800

	// maxVaultAncestorDepth bounds lazy ancestor resolution for files whose
	// parent chain is not materialized (flattened/search listings).
	maxVaultAncestorDepth = 64
)

// vaultRootID returns the private-space root folder ID owned by ownerID, or 0.
func (f *DBFS) vaultRootID(ctx context.Context, ownerID int) int {
	if f.user != nil && ownerID == f.user.ID {
		return f.user.VaultFolder
	}
	if f.userClient == nil {
		return 0
	}
	u, err := f.userClient.GetByID(ctx, ownerID)
	if err != nil || u == nil {
		return 0
	}
	return u.VaultFolder
}

// vaultUnlocked reports whether the current user's vault session is active.
func (f *DBFS) vaultUnlocked() bool {
	if f.cache == nil || f.user == nil || f.user.VaultFolder <= 0 {
		return false
	}
	_, ok := f.cache.Get(VaultUnlockCachePrefix + strconv.Itoa(f.user.ID))
	return ok
}

// chainInVault checks the materialized ancestor chain only.
func chainInVault(f *File, rootID int) bool {
	if rootID <= 0 || f.Model.ID == rootID {
		return false
	}
	for p := f.Parent; p != nil && p.Model != nil; p = p.Parent {
		if p.Model.ID == rootID {
			return true
		}
	}
	return false
}

// fileInVault reports whether the file sits inside the vault rooted at
// rootID. When the parent chain is not materialized it walks file_children
// links upward in bounded steps.
func (f *DBFS) fileInVault(ctx context.Context, file *File, rootID int) (bool, error) {
	if rootID <= 0 || file == nil || file.Model == nil || file.Model.ID == rootID {
		return false, nil
	}
	if file.Parent != nil || file.Model.FileChildren == 0 {
		return chainInVault(file, rootID), nil
	}

	pid := file.Model.FileChildren
	for depth := 0; pid > 0 && depth < maxVaultAncestorDepth; depth++ {
		if pid == rootID {
			return true, nil
		}
		parents, _, err := f.fileClient.GetByIDs(ctx, []int{pid}, 0)
		if err != nil {
			return false, err
		}
		if len(parents) == 0 {
			return false, nil
		}
		pid = parents[0].FileChildren
	}
	return pid == rootID, nil
}

// requireVaultUnlocked allows the operation only for the vault owner with an
// active unlock session; non-owners get a not-found error so vault contents
// are not revealed.
func (f *DBFS) requireVaultUnlocked(file *File) error {
	if f.user.ID == file.Model.OwnerID {
		if f.vaultUnlocked() {
			return nil
		}
		return ErrVaultLocked
	}
	return fs.ErrPathNotExist
}

// requireVaultAccess gates access to the file itself: only files strictly
// inside the vault are gated, the vault root stays resolvable so it can serve
// as the unlock entry point.
func (f *DBFS) requireVaultAccess(ctx context.Context, file *File) error {
	if file == nil || file.Model == nil {
		return nil
	}
	inside, err := f.fileInVault(ctx, file, f.vaultRootID(ctx, file.Model.OwnerID))
	if err != nil || !inside {
		return err
	}
	return f.requireVaultUnlocked(file)
}

// requireVaultEntry gates navigation into a folder: the vault root and
// everything inside it require an unlocked vault.
func (f *DBFS) requireVaultEntry(ctx context.Context, file *File) error {
	if file == nil || file.Model == nil {
		return nil
	}
	if file.Model.ID == f.vaultRootID(ctx, file.Model.OwnerID) {
		return f.requireVaultUnlocked(file)
	}
	return f.requireVaultAccess(ctx, file)
}

// IsInPrivateSpace implements fs.FileSystem.
func (f *DBFS) IsInPrivateSpace(ctx context.Context, file fs.File) (bool, error) {
	dbfsFile, ok := file.(*File)
	if !ok || dbfsFile == nil || dbfsFile.Model == nil {
		return false, nil
	}
	return f.fileInVault(ctx, dbfsFile, f.vaultRootID(ctx, dbfsFile.Model.OwnerID))
}

// vaultNavigator wraps any Navigator and gates access to files located inside
// the file owner's private space. The vault is a regular folder at the user's
// root; a file is "in vault" when its ancestor chain contains the vault
// folder ID.
type vaultNavigator struct {
	Navigator
	fs    *DBFS
	roots map[int]int // ownerID -> vault folder ID, resolved lazily per request
}

func (v *vaultNavigator) vaultRootID(ctx context.Context, ownerID int) int {
	if id, ok := v.roots[ownerID]; ok {
		return id
	}
	id := v.fs.vaultRootID(ctx, ownerID)
	v.roots[ownerID] = id
	return id
}

func (v *vaultNavigator) To(ctx context.Context, path *fs.URI) (*File, error) {
	file, err := v.Navigator.To(ctx, path)
	// To returns the deepest existing ancestor together with NotFound for
	// missing targets — preserve that partial result, but still gate it: the
	// ancestor itself may sit inside a locked vault.
	if file != nil && file.Model != nil {
		if verr := v.fs.requireVaultAccess(ctx, file); verr != nil {
			return nil, verr
		}
	}
	return file, err
}

func (v *vaultNavigator) Children(ctx context.Context, parent *File, args *ListArgs) (*ListResult, error) {
	if parent != nil && parent.Model != nil {
		if parent.Model.ID == v.vaultRootID(ctx, parent.Model.OwnerID) {
			if err := v.fs.requireVaultUnlocked(parent); err != nil {
				return nil, err
			}
		} else if err := v.fs.requireVaultAccess(ctx, parent); err != nil {
			return nil, err
		}
	}
	res, err := v.Navigator.Children(ctx, parent, args)
	if err != nil {
		return nil, err
	}
	filtered, err := v.filterVaulted(ctx, res.Files)
	if err != nil {
		return nil, err
	}
	res.Files = filtered
	return res, nil
}

// filterVaulted removes vaulted files the current user may not see from a
// listing, preserving order. Files without a materialized parent chain are
// resolved in batched levels to keep the query count bounded.
func (v *vaultNavigator) filterVaulted(ctx context.Context, files []*File) ([]*File, error) {
	drop := make(map[*File]bool)
	pending := make(map[*File]int) // file -> ancestor ID currently under examination
	rootOf := make(map[*File]int)

	for _, f := range files {
		if f == nil || f.Model == nil {
			continue
		}
		rootID := v.vaultRootID(ctx, f.Model.OwnerID)
		rootOf[f] = rootID
		if rootID <= 0 || f.Model.ID == rootID {
			continue
		}
		if f.Parent != nil || f.Model.FileChildren == 0 {
			drop[f] = chainInVault(f, rootID) && !v.accessible(f)
			continue
		}
		pending[f] = f.Model.FileChildren
	}

	for depth := 0; len(pending) > 0 && depth < maxVaultAncestorDepth; depth++ {
		// Fast path: pending ancestor equal to the file's vault root resolves
		// without a query.
		ids := make(map[int]bool)
		for f, pid := range pending {
			if pid == rootOf[f] {
				drop[f] = !v.accessible(f)
				delete(pending, f)
				continue
			}
			ids[pid] = true
		}
		if len(pending) == 0 {
			break
		}

		parents, err := v.getAllByIDs(ctx, lo.Keys(ids))
		if err != nil {
			return nil, err
		}
		for f, pid := range pending {
			p, ok := parents[pid]
			if !ok || p.FileChildren == 0 {
				delete(pending, f)
				continue
			}
			pending[f] = p.FileChildren
		}
	}

	// Anything still unresolved past the depth cap is dropped fail-closed.
	for f := range pending {
		drop[f] = true
	}

	res := make([]*File, 0, len(files))
	for _, f := range files {
		if !drop[f] {
			res = append(res, f)
		}
	}
	return res, nil
}

// accessible reports whether the current user may access a vaulted file.
func (v *vaultNavigator) accessible(f *File) bool {
	return v.fs.user.ID == f.Model.OwnerID && v.fs.vaultUnlocked()
}

// getAllByIDs fetches all given IDs, following GetByIDs pagination.
func (v *vaultNavigator) getAllByIDs(ctx context.Context, ids []int) (map[int]*ent.File, error) {
	res := make(map[int]*ent.File, len(ids))
	for page := 0; page >= 0; {
		files, next, err := v.fs.fileClient.GetByIDs(ctx, ids, page)
		if err != nil {
			return nil, err
		}
		for _, fm := range files {
			res[fm.ID] = fm
		}
		page = next
	}
	return res, nil
}

func (v *vaultNavigator) Walk(ctx context.Context, levelFiles []*File, limit, depth int, fn WalkFunc) error {
	for _, f := range levelFiles {
		if f == nil || f.Model == nil {
			continue
		}
		if err := v.fs.requireVaultEntry(ctx, f); err != nil {
			return err
		}
	}
	return v.Navigator.Walk(ctx, levelFiles, limit, depth, fn)
}

func (v *vaultNavigator) ExecuteHook(ctx context.Context, hookType fs.HookType, file *File) error {
	if err := v.fs.requireVaultAccess(ctx, file); err != nil {
		return err
	}
	return v.Navigator.ExecuteHook(ctx, hookType, file)
}
