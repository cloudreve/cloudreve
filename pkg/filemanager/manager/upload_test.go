package manager

import (
	"context"
	"testing"
	"time"

	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
)

func TestMergeByteRanges(t *testing.T) {
	tests := []struct {
		name   string
		ranges [][2]int64
		start  int64
		end    int64
		want   [][2]int64
	}{
		{"empty", nil, 0, 10, [][2]int64{{0, 10}}},
		{"append gap", [][2]int64{{0, 10}}, 20, 30, [][2]int64{{0, 10}, {20, 30}}},
		{"prepend gap", [][2]int64{{20, 30}}, 0, 10, [][2]int64{{0, 10}, {20, 30}}},
		{"adjacent merge", [][2]int64{{0, 10}}, 10, 20, [][2]int64{{0, 20}}},
		{"bridge two", [][2]int64{{0, 10}, {20, 30}}, 10, 20, [][2]int64{{0, 30}}},
		{"contained", [][2]int64{{0, 30}}, 5, 15, [][2]int64{{0, 30}}},
		{"extend right", [][2]int64{{0, 10}}, 5, 20, [][2]int64{{0, 20}}},
		{"extend left", [][2]int64{{10, 20}}, 0, 15, [][2]int64{{0, 20}}},
		{"middle gap", [][2]int64{{0, 5}, {30, 40}}, 10, 20, [][2]int64{{0, 5}, {10, 20}, {30, 40}}},
		{"merge all", [][2]int64{{0, 5}, {10, 15}, {20, 25}}, 4, 21, [][2]int64{{0, 25}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeByteRanges(tt.ranges, tt.start, tt.end)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestRangesCoverFull(t *testing.T) {
	if !rangesCoverFull([][2]int64{{0, 100}}, 100) {
		t.Fatal("expected full coverage")
	}
	if rangesCoverFull([][2]int64{{0, 99}}, 100) {
		t.Fatal("partial coverage reported as full")
	}
	if rangesCoverFull([][2]int64{{0, 50}, {50, 100}}, 100) {
		t.Fatal("unmerged intervals should not report full coverage")
	}
	if rangesCoverFull(nil, 100) {
		t.Fatal("empty intervals reported as full")
	}
}

func newRangedSession(id string, size int64) fs.UploadSession {
	return fs.UploadSession{
		Props: &fs.UploadProps{
			UploadSessionID: id,
			Size:            size,
			ExpireAt:        time.Now().Add(time.Hour),
		},
	}
}

func TestMarkRangeUploaded(t *testing.T) {
	ctx := context.Background()
	m := &manager{kv: cache.NewMemoStore("", nil)}
	session := newRangedSession("test-range-session", 100)
	if err := m.kv.Set(UploadSessionCachePrefix+"test-range-session", session, 60); err != nil {
		t.Fatal(err)
	}

	// Out-of-order arrival: second half first.
	all, err := m.MarkRangeUploaded(ctx, &session, 50, 50)
	if err != nil {
		t.Fatal(err)
	}
	if all {
		t.Fatal("reported complete after only second half")
	}

	all, err = m.MarkRangeUploaded(ctx, &session, 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	if !all {
		t.Fatal("expected complete after full coverage")
	}

	// Session record carries merged coverage.
	raw, ok := m.kv.Get(UploadSessionCachePrefix + "test-range-session")
	if !ok {
		t.Fatal("session missing from KV")
	}
	stored := raw.(fs.UploadSession)
	if len(stored.RangesReceived) != 1 || stored.RangesReceived[0] != [2]int64{0, 100} {
		t.Fatalf("unexpected ranges: %v", stored.RangesReceived)
	}
}

func TestMarkRangeUploadedGap(t *testing.T) {
	ctx := context.Background()
	m := &manager{kv: cache.NewMemoStore("", nil)}
	session := newRangedSession("test-gap-session", 100)
	if err := m.kv.Set(UploadSessionCachePrefix+"test-gap-session", session, 60); err != nil {
		t.Fatal(err)
	}

	for _, r := range [][2]int64{{0, 40}, {60, 100}} {
		all, err := m.MarkRangeUploaded(ctx, &session, r[0], r[1]-r[0])
		if err != nil {
			t.Fatal(err)
		}
		if all {
			t.Fatal("reported complete with a gap in coverage")
		}
	}
}

func TestMarkRangeUploadedMissingSession(t *testing.T) {
	ctx := context.Background()
	m := &manager{kv: cache.NewMemoStore("", nil)}
	session := newRangedSession("gone", 100)

	all, err := m.MarkRangeUploaded(ctx, &session, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if all {
		t.Fatal("missing session reported complete")
	}
}
