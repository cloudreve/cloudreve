package dbfs

import (
	"context"
	"path"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/pkg/activity"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth/requestinfo"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/eventhub"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/samber/lo"
)

// record writes an audit event for the operation. Best-effort: nil-safe
// for stateless/test FS instances and never fails the caller.
func (f *DBFS) record(ctx context.Context, eventType int, opts ...activity.Opt) {
	if f.activityClient == nil {
		return
	}
	activity.Record(ctx, f.settingClient, f.activityClient, eventType, opts...)
}

func (f *DBFS) emitFileCreated(ctx context.Context, file *File) {
	subscribers := f.getEligibleSubscriber(ctx, file, true)
	for _, subscriber := range subscribers {
		subscriber.Publish(eventhub.Event{
			Type:   eventhub.EventTypeCreate,
			FileID: hashid.EncodeFileID(f.hasher, file.Model.ID),
			From:   subscriber.relativePath(file),
		})
	}
}

func (f *DBFS) emitFileModified(ctx context.Context, file *File) {
	subscribers := f.getEligibleSubscriber(ctx, file, true)
	for _, subscriber := range subscribers {
		subscriber.Publish(eventhub.Event{
			Type:   eventhub.EventTypeModify,
			FileID: hashid.EncodeFileID(f.hasher, file.Model.ID),
			From:   subscriber.relativePath(file),
		})
	}
}

func (f *DBFS) emitFileRenamed(ctx context.Context, file *File, newName string) {
	subscribers := f.getEligibleSubscriber(ctx, file, true)
	for _, subscriber := range subscribers {
		from := subscriber.relativePath(file)
		to := strings.TrimSuffix(from, file.Name()) + newName
		subscriber.Publish(eventhub.Event{
			Type:   eventhub.EventTypeRename,
			FileID: hashid.EncodeFileID(f.hasher, file.Model.ID),
			From:   subscriber.relativePath(file),
			To:     to,
		})
	}
}

func (f *DBFS) emitFileDeleted(ctx context.Context, files ...*File) {
	for _, file := range files {
		subscribers := f.getEligibleSubscriber(ctx, file, true)
		for _, subscriber := range subscribers {
			subscriber.Publish(eventhub.Event{
				Type:   eventhub.EventTypeDelete,
				FileID: hashid.EncodeFileID(f.hasher, file.Model.ID),
				From:   subscriber.relativePath(file),
			})
		}
	}
}

func (f *DBFS) emitFileMoved(ctx context.Context, src, dst *File) {
	srcSubMap := lo.SliceToMap(f.getEligibleSubscriber(ctx, src, true), func(subscriber foundSubscriber) (string, *foundSubscriber) {
		return subscriber.ID(), &subscriber
	})
	dstSubMap := lo.SliceToMap(f.getEligibleSubscriber(ctx, dst, false), func(subscriber foundSubscriber) (string, *foundSubscriber) {
		return subscriber.ID(), &subscriber
	})

	for _, subscriber := range srcSubMap {
		subId := subscriber.ID()
		if dstSub, ok := dstSubMap[subId]; ok {
			// Src and Dst subscribed by the same subscriber
			subscriber.Publish(eventhub.Event{
				Type:   eventhub.EventTypeRename,
				FileID: hashid.EncodeFileID(f.hasher, src.Model.ID),
				From:   subscriber.relativePath(src),
				To:     path.Join(dstSub.relativePath(dst), src.Name()),
			})
			delete(dstSubMap, subId)
		} else {
			// Only Src is subscribed by the subscriber
			subscriber.Publish(eventhub.Event{
				Type:   eventhub.EventTypeDelete,
				FileID: hashid.EncodeFileID(f.hasher, src.Model.ID),
				From:   subscriber.relativePath(src),
			})
		}
	}

	for _, subscriber := range dstSubMap {
		// Only Dst is subscribed by the subscriber
		subscriber.Publish(eventhub.Event{
			Type:   eventhub.EventTypeCreate,
			FileID: hashid.EncodeFileID(f.hasher, src.Model.ID),
			From:   path.Join(subscriber.relativePath(dst), src.Name()),
		})
	}

}

func (f *DBFS) getEligibleSubscriber(ctx context.Context, file *File, checkParentPerm bool) []foundSubscriber {
	roots := file.Ancestors()
	if !checkParentPerm {
		// Include file itself
		roots = file.AncestorsChain()
	}
	requestInfo := requestinfo.RequestInfoFromContext(ctx)
	eligibleSubscribers := make([]foundSubscriber, 0)

	for _, root := range roots {
		subscribers := f.eventHub.GetSubscribers(ctx, root.Model.ID)
		subscribers = lo.Filter(subscribers, func(subscriber eventhub.Subscriber, index int) bool {
			// Exlucde self from subscribers
			if requestInfo != nil && subscriber.ID() == requestInfo.ClientID {
				return false
			}
			return true
		})
		eligibleSubscribers = append(eligibleSubscribers, lo.Map(subscribers, func(subscriber eventhub.Subscriber, index int) foundSubscriber {
			return foundSubscriber{
				Subscriber: subscriber,
				root:       root,
			}
		})...)
	}

	return eligibleSubscribers

}

type foundSubscriber struct {
	eventhub.Subscriber
	root *File
}

func (s *foundSubscriber) relativePath(file *File) string {
	res := strings.TrimPrefix(file.Uri(true).Path(), s.root.Uri(true).Path())
	if res == "" {
		res = fs.Separator
	}

	if res[0] != fs.Separator[0] {
		res = fs.Separator + res
	}

	return res
}
