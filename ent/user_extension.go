package ent

import (
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/samber/lo"
)

func (u *User) AnyGroup(predict func(*Group) bool) bool {
	return !lo.NoneBy(u.Edges.Group, predict)
}

func (u *User) EnforceGroupPermission(perm types.GroupPermission) bool {
	return !lo.NoneBy(u.Edges.Group, func(item *Group) bool {
		return item.Permissions.Enabled(int(perm))
	})
}

func (u *User) GroupMaxStorage() int64 {
	return lo.MaxBy(u.Edges.Group, func(a *Group, b *Group) bool {
		return a.MaxStorage > b.MaxStorage
	}).MaxStorage
}

func (u *User) GroupMaxWalkedFiles() int {
	return max(lo.MaxBy(u.Edges.Group, func(a *Group, b *Group) bool {
		return a.Settings.MaxWalkedFiles > b.Settings.MaxWalkedFiles
	}).Settings.MaxWalkedFiles, 1)
}

func (u *User) GroupMaxTrashRetention() int {
	return lo.MaxBy(u.Edges.Group, func(a *Group, b *Group) bool {
		return a.Settings.TrashRetention > b.Settings.TrashRetention
	}).Settings.TrashRetention
}

func (u *User) GroupMaxDecompressSize() int64 {
	return lo.MaxBy(u.Edges.Group, func(a *Group, b *Group) bool {
		return a.Settings.DecompressSize > b.Settings.DecompressSize
	}).Settings.DecompressSize
}

func (u *User) GroupMaxCompressSize() int64 {
	return lo.MaxBy(u.Edges.Group, func(a *Group, b *Group) bool {
		return a.Settings.CompressSize > b.Settings.CompressSize
	}).Settings.CompressSize
}

func (u *User) GroupMaxSpeedLimit() int {
	return lo.MaxBy(u.Edges.Group, func(a *Group, b *Group) bool {
		return a.SpeedLimit > b.SpeedLimit
	}).SpeedLimit
}

func (u *User) GroupMaxSourceBatchSize() int {
	return lo.MaxBy(u.Edges.Group, func(a *Group, b *Group) bool {
		return a.Settings.SourceBatchSize > b.Settings.SourceBatchSize
	}).Settings.SourceBatchSize
}

func (u *User) GroupMaxAria2BatchSize() int {
	return lo.MaxBy(u.Edges.Group, func(a *Group, b *Group) bool {
		return a.Settings.Aria2BatchSize > b.Settings.Aria2BatchSize
	}).Settings.Aria2BatchSize
}

func (u *User) GetPrimaryGroup() *Group {
	g, ok := lo.Find(u.Edges.Membership, func(d *Membership) bool {
		return d.IsPrimary
	})

	if ok {
		return g.Edges.Group
	}

	if len(u.Edges.Group) > 0 {
		return u.Edges.Group[0]
	}

	return nil
}
