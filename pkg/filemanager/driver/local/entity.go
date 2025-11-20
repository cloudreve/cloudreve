package local

import (
	"os"
	"time"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
)

// NewLocalFileEntity creates a new local file entity.
func NewLocalFileEntity(t types.EntityType, src string) (fs.Entity, error) {
	info, err := os.Stat(util.RelativePath(src))
	if err != nil {
		return nil, err
	}

	return &localFileEntity{
		t:    t,
		src:  src,
		size: info.Size(),
	}, nil
}

type localFileEntity struct {
	t    types.EntityType
	src  string
	size int64
}

func (l *localFileEntity) ID() int {
	return 0
}

func (l *localFileEntity) Type() types.EntityType {
	return l.t
}

func (l *localFileEntity) Size() int64 {
	return l.size
}

func (l *localFileEntity) UpdatedAt() time.Time {
	return time.Now()
}

func (l *localFileEntity) CreatedAt() time.Time {
	return time.Now()
}

func (l *localFileEntity) CreatedBy() *ent.User {
	return nil
}

func (l *localFileEntity) Source() string {
	return l.src
}

func (l *localFileEntity) ReferenceCount() int {
	return 1
}

func (l *localFileEntity) PolicyID() int {
	return 0
}

func (l *localFileEntity) UploadSessionID() *uuid.UUID {
	return nil
}

func (l *localFileEntity) Model() *ent.Entity {
	return nil
}

func (l *localFileEntity) Props() *types.EntityProps {
	return nil
}

func (l *localFileEntity) Encrypted() bool {
	return false
}
