package explorer

import (
    "context"
    "errors"
    "fmt"

    "github.com/cloudreve/Cloudreve/v4/application/dependency"
    "github.com/cloudreve/Cloudreve/v4/inventory"
    "github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
    "github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
    "github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager"
    "github.com/cloudreve/Cloudreve/v4/pkg/serializer"
    "github.com/cloudreve/Cloudreve/v4/pkg/thumb"
)

type (
	// ResetThumbParamCtx defines context for ResetThumbService
	ResetThumbParamCtx struct{}

	// ResetThumbService handles resetting thumbnail generation for files
	ResetThumbService struct {
		Uris []string `json:"uris" binding:"required,min=1"`
	}

	// ResetThumbError represents an error for a specific URI
	ResetThumbError struct {
		URI    string `json:"uri"`
		Reason string `json:"reason"`
	}

	// ResetThumbResponse represents the response for reset thumbnail operation
	ResetThumbResponse struct {
		Errors []ResetThumbError `json:"errors,omitempty"`
	}
)

func (s *ResetThumbService) GetUris() []string {
	return s.Uris
}

// Reset resets thumbnail generation for the specified files
func (s *ResetThumbService) Reset(c context.Context) (*ResetThumbResponse, error) {
    dep := dependency.FromContext(c)
    user := inventory.UserFromContext(c)
    m := manager.NewFileManager(dep, user)
    defer m.Recycle()

	uris, err := fs.NewUriFromStrings(s.Uris...)
	if err != nil {
		return nil, serializer.NewError(serializer.CodeParamErr, "unknown uri", err)
	}

    var errs []ResetThumbError

    for _, uri := range uris {
        // Get the file to check if it exists and get current metadata
        file, err := m.Get(c, uri, dbfs.WithFilePublicMetadata())
        if err != nil {
            errs = append(errs, ResetThumbError{
                URI:    uri.String(),
                Reason: fmt.Sprintf("Reset thumbnail error: File does not exist"),
            })
            continue
        }

        // Check if thumb:disabled metadata exists
        metadata := file.Metadata()
        if _, exists := metadata[dbfs.ThumbDisabledKey]; exists {
            // Remove the disabled mark directly via FileClient to bypass metadata validation
            fileClient := dep.FileClient()
            if dbfsFile, ok := file.(*dbfs.File); ok {
                if err := fileClient.RemoveMetadata(c, dbfsFile.Model, dbfs.ThumbDisabledKey); err != nil {
                    errs = append(errs, ResetThumbError{
                        URI:    uri.String(),
                        Reason: fmt.Sprintf("Reset thumbnail error: Failed to update metadata - %s", err.Error()),
                    })
                    continue
                }
            } else {
                errs = append(errs, ResetThumbError{
                    URI:    uri.String(),
                    Reason: "Reset thumbnail error: Failed to access file model",
                })
                continue
            }
        }

        // Trigger thumbnail regeneration by accessing the thumbnail
        _, err = m.Thumbnail(c, uri)
        if err != nil {
            // Unsupported or not available
            if errors.Is(err, thumb.ErrNotAvailable) || errors.Is(err, fs.ErrEntityNotExist) {
                errs = append(errs, ResetThumbError{
                    URI:    uri.String(),
                    Reason: "Reset thumbnail error: File does not support thumbnail generation",
                })
            } else {
                errs = append(errs, ResetThumbError{
                    URI:    uri.String(),
                    Reason: fmt.Sprintf("Reset thumbnail error: Failed to generate thumbnail - %s", err.Error()),
                })
            }
        }
    }

    if len(errs) > 0 {
        return &ResetThumbResponse{Errors: errs}, nil
    }

	return &ResetThumbResponse{}, nil
}

// (Thumb ext list API removed; use site config section "thumb")
