package manager

import (
	"context"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
)

// RelocateBlobArgs args to relocate the blob of an entity into another storage policy.
type RelocateBlobArgs struct {
	// Entity whose blob is being relocated.
	Entity fs.Entity
	// Uri of the file owning the entity. Storage drivers fall back to it for MIME type
	// detection, so it should be given whenever the owning file is known.
	Uri *fs.URI
	// DstPolicy is the storage policy the blob is relocated into.
	DstPolicy *ent.StoragePolicy
	// SavePath is the path of the blob in the destination storage policy.
	SavePath string
	// ProgressFunc, if given, is called while the blob is being transferred.
	ProgressFunc fs.ProgressFunc
}

// RelocateBlob copies the blob of args.Entity into args.DstPolicy at args.SavePath, applying the
// file encryption setting of the destination policy along the way.
//
// A blob that is already encrypted is transferred in its ciphertext form, keeping the data key it
// was encrypted with, so that relocation costs no crypto and never invalidates the key stored on
// the entity. A plaintext blob relocated into a policy with encryption enabled is encrypted during
// the transfer under a freshly generated data key.
//
// When not nil, the returned metadata describes the encryption of the relocated blob and has to be
// persisted onto the entity once the caller commits the relocation; a nil return means the
// encryption state of the entity is left as it is. The metadata is returned rather than persisted
// here so that it never needs to be carried in a resumable task state, where the plaintext data
// key would end up written into the database.
func (m *manager) RelocateBlob(ctx context.Context, args *RelocateBlobArgs) (*types.EncryptMetadata, error) {
	if args == nil || args.Entity == nil || args.DstPolicy == nil {
		return nil, fmt.Errorf("invalid relocate blob arguments")
	}

	es, err := m.GetEntitySource(ctx, 0, fs.WithEntity(args.Entity))
	if err != nil {
		return nil, fmt.Errorf("failed to open source entity: %w", err)
	}
	defer es.Close()

	var metadata *types.EncryptMetadata
	if args.Entity.Encrypted() {
		// Present the blob as stored, the data key travels with the entity row.
		es.Apply(entitysource.WithDisableCryptor())
	} else if args.DstPolicy.Settings != nil && args.DstPolicy.Settings.Encryption {
		cryptor, err := m.dep.EncryptorFactory(ctx)(types.CipherAES256CTR)
		if err != nil {
			return nil, fmt.Errorf("failed to create cryptor: %w", err)
		}

		metadata, err = cryptor.GenerateMetadata(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to generate encrypt metadata: %w", err)
		}
	}

	req := &fs.UploadRequest{
		Props: &fs.UploadProps{
			Uri:      args.Uri,
			Size:     args.Entity.Size(),
			SavePath: args.SavePath,
		},
		Mode:         fs.ModeOverwrite,
		File:         es,
		Seeker:       es,
		ProgressFunc: args.ProgressFunc,
	}

	if args.Uri != nil {
		req.Props.MimeType = m.dep.MimeDetector(ctx).TypeByName(args.Uri.Name())
	}

	if metadata != nil {
		// The stream is encrypted from its very beginning, hence a zero counter offset. AES-256-CTR
		// preserves the length of the stream, so Props.Size stays valid for the ciphertext and the
		// size recorded on the entity needs no adjustment.
		if err := encryptUploadRequest(ctx, m.dep.EncryptorFactory(ctx), req, metadata, 0); err != nil {
			return nil, err
		}
	}

	d, err := m.GetStorageDriver(ctx, m.CastStoragePolicyOnSlave(ctx, args.DstPolicy))
	if err != nil {
		return nil, err
	}

	if err := d.Put(ctx, req); err != nil {
		return nil, serializer.NewError(serializer.CodeIOFailed, "Failed to relocate blob", err)
	}

	if metadata == nil {
		return nil, nil
	}

	// The plaintext data key is dropped from the returned metadata: it is only needed while the
	// stream is being encrypted, whereas what is returned here is meant to be persisted, on the
	// entity and possibly in a resumable task state along the way.
	return &types.EncryptMetadata{
		Algorithm: metadata.Algorithm,
		Key:       metadata.Key,
		IV:        metadata.IV,
	}, nil
}
