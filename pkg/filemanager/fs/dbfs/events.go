package dbfs

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/auth/requestinfo"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/eventhub"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/samber/lo"
)

func (f *DBFS) emitFileCreated(ctx context.Context, file *File) {
	f.emitFileEvent(ctx, eventhub.EventTypeCreate, file)
}

func (f *DBFS) emitFileModified(ctx context.Context, file *File) {
	f.emitFileEvent(ctx, eventhub.EventTypeModify, file)
}

func (f *DBFS) emitFileRenamed(ctx context.Context, file *File, newName string) {
	file = snapshotEventFile(file)
	f.emitAfterCommit(ctx, func(ctx context.Context) {
		subscribers := f.getEligibleSubscriber(ctx, file, true)
		for _, subscriber := range subscribers {
			from := subscriber.relativePath(file)
			to := strings.TrimSuffix(from, file.Name()) + newName
			subscriber.Publish(eventhub.Event{
				Type:   eventhub.EventTypeRename,
				FileID: hashid.EncodeFileID(f.hasher, file.Model.ID),
				From:   from,
				To:     to,
			})
		}
	})
}

func (f *DBFS) emitFileDeleted(ctx context.Context, files ...*File) {
	for _, file := range files {
		f.emitFileEvent(ctx, eventhub.EventTypeDelete, file)
	}
}

func (f *DBFS) emitFileEvent(ctx context.Context, eventType eventhub.EventType, file *File) {
	file = snapshotEventFile(file)
	f.emitAfterCommit(ctx, func(ctx context.Context) {
		subscribers := f.getEligibleSubscriber(ctx, file, true)
		for _, subscriber := range subscribers {
			subscriber.Publish(eventhub.Event{
				Type:   eventType,
				FileID: hashid.EncodeFileID(f.hasher, file.Model.ID),
				From:   subscriber.relativePath(file),
			})
		}
	})
}

func (f *DBFS) emitFileMoved(ctx context.Context, src, dst *File) {
	src, dst = snapshotEventFile(src), snapshotEventFile(dst)
	f.emitAfterCommit(ctx, func(ctx context.Context) {
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
	})
}

func (f *DBFS) emitAfterCommit(ctx context.Context, emit func(context.Context)) {
	inventory.AfterCommit(ctx, func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		emit(ctx)
	})
}

// Files are pooled and can be replaced before an enclosing transaction commits.
func snapshotEventFile(file *File) *File {
	if file == nil {
		return nil
	}
	model := *file.Model
	snapshot := &File{
		Model:      &model,
		Parent:     snapshotEventFile(file.Parent),
		OwnerModel: file.Owner(),
	}
	for i, isRoot := range []bool{true, false} {
		if uri := file.Uri(isRoot); uri != nil {
			u := *uri.U
			snapshot.Path[i] = &fs.URI{U: &u}
		}
	}
	return snapshot
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
