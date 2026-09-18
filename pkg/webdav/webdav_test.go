package webdav

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/boolset"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
)

type copyMoveOperationsStub struct {
	calls      []string
	deleteErr  error
	moveErr    error
	renameErr  error
	moveIsCopy bool
}

func (s *copyMoveOperationsStub) Delete(context.Context, []*fs.URI, ...fs.Option) error {
	s.calls = append(s.calls, "delete")
	return s.deleteErr
}

func (s *copyMoveOperationsStub) MoveOrCopy(_ context.Context, _ []*fs.URI, _ *fs.URI, isCopy bool) error {
	s.calls = append(s.calls, "move")
	s.moveIsCopy = isCopy
	return s.moveErr
}

func (s *copyMoveOperationsStub) Rename(context.Context, *fs.URI, string) (fs.File, error) {
	s.calls = append(s.calls, "rename")
	return nil, s.renameErr
}

func TestParseOverwrite(t *testing.T) {
	tests := []struct {
		value string
		want  bool
		err   bool
	}{
		{value: "", want: true},
		{value: "T", want: true},
		{value: "F", want: false},
		{value: "true", err: true},
	}

	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			got, err := parseOverwrite(test.value)
			if (err != nil) != test.err {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("unexpected overwrite value: %v", got)
			}
		})
	}
}

func TestPerformCopyMove(t *testing.T) {
	src := mustWebDAVTestURI(t, "cloudreve://my/source.temp")
	dst := mustWebDAVTestURI(t, "cloudreve://my/source.txt")
	dstFolder := dst.DirUri()

	t.Run("overwrite disabled", func(t *testing.T) {
		operations := &copyMoveOperationsStub{}
		status, err := performCopyMove(context.Background(), operations, src, dst, dstFolder, false, false, true)
		if status != http.StatusPreconditionFailed || !errors.Is(err, errDestinationExists) {
			t.Fatalf("unexpected result: status=%d err=%v", status, err)
		}
		if len(operations.calls) != 0 {
			t.Fatalf("unexpected operations: %v", operations.calls)
		}
	})

	t.Run("overwrite existing move", func(t *testing.T) {
		operations := &copyMoveOperationsStub{}
		status, err := performCopyMove(context.Background(), operations, src, dst, dstFolder, false, true, true)
		if err != nil || status != http.StatusNoContent {
			t.Fatalf("unexpected result: status=%d err=%v", status, err)
		}
		if got := operations.calls; len(got) != 3 || got[0] != "delete" || got[1] != "move" || got[2] != "rename" {
			t.Fatalf("unexpected operation order: %v", got)
		}
		if operations.moveIsCopy {
			t.Fatal("move was executed as copy")
		}
	})

	t.Run("new copy", func(t *testing.T) {
		operations := &copyMoveOperationsStub{}
		status, err := performCopyMove(context.Background(), operations, src, dst, dstFolder, true, true, false)
		if err != nil || status != http.StatusCreated {
			t.Fatalf("unexpected result: status=%d err=%v", status, err)
		}
		if got := operations.calls; len(got) != 2 || got[0] != "move" || got[1] != "rename" {
			t.Fatalf("unexpected operation order: %v", got)
		}
		if !operations.moveIsCopy {
			t.Fatal("copy was executed as move")
		}
	})
}

func mustWebDAVTestURI(t *testing.T, raw string) *fs.URI {
	t.Helper()
	uri, err := fs.NewUriFromString(raw)
	if err != nil {
		t.Fatal(err)
	}
	return uri
}

func TestParseContentRange(t *testing.T) {
	tests := []struct {
		header string
		want   *contentRange
		err    bool
	}{
		{"", nil, false},
		{"bytes 0-1048575/3145728", &contentRange{0, 1048575, 3145728}, false},
		{"bytes 1048576-2097151/3145728", &contentRange{1048576, 2097151, 3145728}, false},
		{"bytes 0-99/100", &contentRange{0, 99, 100}, false},
		{" bytes 0-9/10 ", &contentRange{0, 9, 10}, false},
		{"bytes 0-9/*", nil, true},
		{"items 0-9/10", nil, true},
		{"bytes 0-9", nil, true},
		{"bytes */10", nil, true},
		{"bytes 9-0/10", nil, true},
		{"bytes 0-10/10", nil, true},
		{"bytes -1-9/10", nil, true},
		{"bytes 0-9/0", nil, true},
		{"bytes a-b/c", nil, true},
	}

	for _, tt := range tests {
		got, err := parseContentRange(tt.header)
		if tt.err {
			if err == nil {
				t.Fatalf("header %q: expected error, got %+v", tt.header, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("header %q: unexpected error: %v", tt.header, err)
		}
		if tt.want == nil {
			if got != nil {
				t.Fatalf("header %q: expected nil, got %+v", tt.header, got)
			}
			continue
		}
		if got == nil || *got != *tt.want {
			t.Fatalf("header %q: got %+v, want %+v", tt.header, got, tt.want)
		}
	}
}

func TestDavWriteForbidden(t *testing.T) {
	readOnlyAccount := &boolset.BooleanSet{}
	boolset.Set(types.DavAccountReadOnly, true, readOnlyAccount)
	readOnlyGroup := &boolset.BooleanSet{}
	boolset.Set(types.GroupPermissionWebDAVReadOnly, true, readOnlyGroup)

	mkUser := func(group *boolset.BooleanSet, account *boolset.BooleanSet) *ent.User {
		u := &ent.User{}
		if group != nil {
			u.SetGroup(&ent.Group{Permissions: group})
		}
		if account != nil {
			u.Edges.DavAccounts = []*ent.DavAccount{{Options: account}}
		}
		return u
	}

	if !davWriteForbidden(mkUser(nil, readOnlyAccount)) {
		t.Fatal("read-only dav account not blocked")
	}
	if !davWriteForbidden(mkUser(readOnlyGroup, &boolset.BooleanSet{})) {
		t.Fatal("read-only group not blocked")
	}
	if davWriteForbidden(mkUser(&boolset.BooleanSet{}, &boolset.BooleanSet{})) {
		t.Fatal("normal user blocked")
	}
	if davWriteForbidden(nil) {
		t.Fatal("nil user blocked")
	}
}
