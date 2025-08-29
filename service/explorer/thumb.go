package explorer

import (
    "context"
    "errors"
    "fmt"
    "sort"
    "strings"

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

type (
	// ThumbExtsParamCtx defines context for ThumbExtsService
	ThumbExtsParamCtx struct{}

	// ThumbExtsService handles getting supported thumbnail extensions
	ThumbExtsService struct{}

	// ThumbExtsResponse represents the response for supported extensions
	ThumbExtsResponse struct {
		Exts []string `json:"exts"`
	}
)

// Get returns all supported thumbnail extensions from enabled generators
func (s *ThumbExtsService) Get(c context.Context) (*ThumbExtsResponse, error) {
    dep := dependency.FromContext(c)
    settings := dep.SettingProvider()

    extensions := make(map[string]bool)

	// Built-in generator (always supports these if enabled)
	if settings.BuiltinThumbGeneratorEnabled(c) {
		for _, ext := range []string{"jpg", "jpeg", "png", "gif"} {
			extensions[ext] = true
		}
	}

    // FFMpeg generator
    if settings.FFMpegThumbGeneratorEnabled(c) {
        for _, ext := range settings.FFMpegThumbExts(c) {
            extensions[strings.ToLower(ext)] = true
        }
    }

    // Vips generator
    if settings.VipsThumbGeneratorEnabled(c) {
        for _, ext := range settings.VipsThumbExts(c) {
            extensions[strings.ToLower(ext)] = true
        }
    }

    // LibreOffice generator
    if settings.LibreOfficeThumbGeneratorEnabled(c) {
        for _, ext := range settings.LibreOfficeThumbExts(c) {
            extensions[strings.ToLower(ext)] = true
        }
    }

    // Music cover generator
    if settings.MusicCoverThumbGeneratorEnabled(c) {
        for _, ext := range settings.MusicCoverThumbExts(c) {
            extensions[strings.ToLower(ext)] = true
        }
    }

    // LibRaw generator
    if settings.LibRawThumbGeneratorEnabled(c) {
        for _, ext := range settings.LibRawThumbExts(c) {
            extensions[strings.ToLower(ext)] = true
        }
    }

	// Convert map to sorted slice
	result := make([]string, 0, len(extensions))
	for ext := range extensions {
		result = append(result, ext)
	}

	// Sort extensions alphabetically using Go's built-in sort
	sort.Strings(result)

	return &ThumbExtsResponse{Exts: result}, nil
}
