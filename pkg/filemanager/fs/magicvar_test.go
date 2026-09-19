package fs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReplaceMagicVar(t *testing.T) {
	a := assert.New(t)
	now := time.Date(2024, 3, 5, 6, 7, 8, 0, time.UTC)
	props := MagicVarProps{
		FsSeparator:      "/",
		PathAvailable:    true,
		BlobAvailable:    true,
		Time:             now,
		UserID:           42,
		OriginName:       "photo.jpg",
		OriginPath:       "album/2024",
		CompleteBlobPath: "blobs/abc123.bin",
	}

	a.Equal("u42", ReplaceMagicVar("u{uid}", props))
	a.Equal("20240305", ReplaceMagicVar("{date}", props))
	a.Equal("20240305060708", ReplaceMagicVar("{datetime}", props))
	a.Equal("2024/03/05", ReplaceMagicVar("{year}/{month}/{day}", props))
	a.Equal("photo.jpg", ReplaceMagicVar("{originname}", props))
	a.Equal("photo", ReplaceMagicVar("{originname_without_ext}", props))
	a.Equal(".jpg", ReplaceMagicVar("{ext}", props))
	a.Equal("album/2024/", ReplaceMagicVar("{path}", props))
	a.Equal("abc123.bin", ReplaceMagicVar("{blob_name}", props))
	a.Equal("abc123", ReplaceMagicVar("{blob_name_without_ext}", props))
	a.Equal("blobs/", ReplaceMagicVar("{blob_path}", props))
	a.Equal("{unknown}", ReplaceMagicVar("{unknown}", props))

	// Unavailable context leaves the placeholder untouched.
	noPath := props
	noPath.PathAvailable = false
	a.Equal("{path}", ReplaceMagicVar("{path}", noPath))
	noBlob := props
	noBlob.BlobAvailable = false
	a.Equal("{blob_name}", ReplaceMagicVar("{blob_name}", noBlob))

	// Random vars produce non-empty output.
	a.Len(ReplaceMagicVar("{randomkey8}", props), 8)
	a.Len(ReplaceMagicVar("{uuid}", props), 36)
}
