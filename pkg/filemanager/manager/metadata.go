package manager

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/application/constants"
	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/fs/dbfs"
	"github.com/cloudreve/Cloudreve/v4/pkg/hashid"
	"github.com/cloudreve/Cloudreve/v4/pkg/serializer"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
)

type (
	metadataValidator func(ctx context.Context, m *manager, patch *fs.MetadataPatch) error
)

const (
	wildcardMetadataKey       = "*"
	customizeMetadataSuffix   = "customize"
	tagMetadataSuffix         = "tag"
	customPropsMetadataSuffix = "props"
	iconColorMetadataKey      = customizeMetadataSuffix + ":icon_color"
	emojiIconMetadataKey      = customizeMetadataSuffix + ":emoji"
	shareOwnerMetadataKey     = dbfs.MetadataSysPrefix + "shared_owner"
	shareRedirectMetadataKey  = dbfs.MetadataSysPrefix + "shared_redirect"
)

var (
	validate = validator.New()

	lastEmojiHash = ""
	emojiPresets  = map[string]struct{}{}

	// validateColor validates a color value
	validateColor = func(optional bool) metadataValidator {
		return func(ctx context.Context, m *manager, patch *fs.MetadataPatch) error {
			patch.UpdateModifiedAt = true

			if patch.Remove {
				return nil
			}

			tag := "omitempty,iscolor"
			if !optional {
				tag = "required,iscolor"
			}

			res := validate.Var(patch.Value, tag)
			if res != nil {
				return fmt.Errorf("invalid color: %w", res)
			}

			return nil
		}
	}
	validators = map[string]map[string]metadataValidator{
		"sys": {
			wildcardMetadataKey: func(ctx context.Context, m *manager, patch *fs.MetadataPatch) error {
				if patch.Remove {
					return fmt.Errorf("cannot remove system metadata")
				}

				patch.UpdateModifiedAt = true

				dep := dependency.FromContext(ctx)
				// Validate share owner is valid hashid
				if patch.Key == shareOwnerMetadataKey {
					hasher := dep.HashIDEncoder()
					_, err := hasher.Decode(patch.Value, hashid.UserID)
					if err != nil {
						return fmt.Errorf("invalid share owner: %w", err)
					}

					return nil
				}

				// Validate share redirect uri is valid share uri
				if patch.Key == shareRedirectMetadataKey {
					uri, err := fs.NewUriFromString(patch.Value)
					if err != nil || uri.FileSystem() != constants.FileSystemShare {
						return fmt.Errorf("invalid redirect uri: %w", err)
					}

					return nil
				}

				return fmt.Errorf("unsupported system metadata key: %s", patch.Key)
			},
		},
		"dav": {},
		// Allow manipulating thumbnail metadata via public PatchMetadata API
		"thumb": {
			// Only supported thumb metadata currently is thumb:disabled
			dbfs.ThumbDisabledKey: func(ctx context.Context, m *manager, patch *fs.MetadataPatch) error {
				// Presence of this key disables thumbnails; value is ignored.
				// We allow both setting and removing this key.
				return nil
			},
		},
		customizeMetadataSuffix: {
			iconColorMetadataKey: validateColor(false),
			emojiIconMetadataKey: func(ctx context.Context, m *manager, patch *fs.MetadataPatch) error {
				patch.UpdateModifiedAt = true

				if patch.Remove {
					return nil
				}

				// Validate if patched emoji is within preset list.
				emojis := m.settings.EmojiPresets(ctx)
				current := fmt.Sprintf("%x", (sha1.Sum([]byte(emojis))))
				if current != lastEmojiHash {
					presets := make(map[string][]string)
					if err := json.Unmarshal([]byte(emojis), &presets); err != nil {
						return fmt.Errorf("failed to read emoji setting: %w", err)
					}

					emojiPresets = make(map[string]struct{})
					for _, v := range presets {
						for _, emoji := range v {
							emojiPresets[emoji] = struct{}{}
						}
					}
				}

				if _, ok := emojiPresets[patch.Value]; !ok {
					return fmt.Errorf("unsupported emoji")
				}
				return nil
			},
		},
		tagMetadataSuffix: {
			wildcardMetadataKey: func(ctx context.Context, m *manager, patch *fs.MetadataPatch) error {
				patch.UpdateModifiedAt = true

				if err := validateColor(true)(ctx, m, patch); err != nil {
					return err
				}

				if patch.Key == tagMetadataSuffix+":" {
					return fmt.Errorf("invalid metadata key")
				}

				return nil
			},
		},
		customPropsMetadataSuffix: {
			wildcardMetadataKey: func(ctx context.Context, m *manager, patch *fs.MetadataPatch) error {
				patch.UpdateModifiedAt = true

				if patch.Remove {
					return nil
				}

				customProps := m.settings.CustomProps(ctx)
				propId := strings.TrimPrefix(patch.Key, customPropsMetadataSuffix+":")
				for _, prop := range customProps {
					if prop.ID == propId {
						switch prop.Type {
						case types.CustomPropsTypeText:
							if prop.Min > 0 && prop.Min > len(patch.Value) {
								return fmt.Errorf("value is too short")
							}
							if prop.Max > 0 && prop.Max < len(patch.Value) {
								return fmt.Errorf("value is too long")
							}

							return nil
						case types.CustomPropsTypeRating:
							if patch.Value == "" {
								return nil
							}

							// validate the value is a number
							rating, err := strconv.Atoi(patch.Value)
							if err != nil {
								return fmt.Errorf("value is not a number")
							}

							if prop.Max < rating {
								return fmt.Errorf("value is too large")
							}

							return nil

						case types.CustomPropsTypeNumber:
							if patch.Value == "" {
								return nil
							}

							value, err := strconv.Atoi(patch.Value)
							if err != nil {
								return fmt.Errorf("value is not a number")
							}

							if prop.Min > value {
								return fmt.Errorf("value is too small")
							}
							if prop.Max > 0 && prop.Max < value {
								return fmt.Errorf("value is too large")
							}

							return nil

						case types.CustomPropsTypeBoolean:
							if patch.Value == "" {
								return nil
							}

							if patch.Value != "true" && patch.Value != "false" {
								return fmt.Errorf("value is not a boolean")
							}

							return nil
						case types.CustomPropsTypeSelect:
							if patch.Value == "" {
								return nil
							}

							for _, option := range prop.Options {
								if option == patch.Value {
									return nil
								}
							}

							return fmt.Errorf("invalid option")
						case types.CustomPropsTypeMultiSelect:
							if patch.Value == "" {
								return nil
							}

							var values []string
							if err := json.Unmarshal([]byte(patch.Value), &values); err != nil {
								return fmt.Errorf("invalid multi select value: %w", err)
							}

							// make sure all values are in the options
							for _, value := range values {
								if !lo.Contains(prop.Options, value) {
									return fmt.Errorf("invalid option")
								}
							}

							return nil

						case types.CustomPropsTypeLink:
							if patch.Value == "" {
								return nil
							}

							if prop.Min > 0 && len(patch.Value) < prop.Min {
								return fmt.Errorf("value is too small")
							}

							if prop.Max > 0 && len(patch.Value) > prop.Max {
								return fmt.Errorf("value is too large")
							}

							return nil
						default:
							return nil
						}
					}
				}

				return fmt.Errorf("unkown custom props")
			},
		},
	}
)

func (m *manager) PatchMedata(ctx context.Context, path []*fs.URI, data ...fs.MetadataPatch) error {
	data, err := m.validateMetadata(ctx, data...)
	if err != nil {
		return err
	}

	return m.fs.PatchMetadata(ctx, path, data...)
}

func (m *manager) validateMetadata(ctx context.Context, data ...fs.MetadataPatch) ([]fs.MetadataPatch, error) {
	validated := make([]fs.MetadataPatch, 0, len(data))
	for _, patch := range data {
		category := strings.Split(patch.Key, ":")
		if len(category) < 2 {
			return validated, serializer.NewError(serializer.CodeParamErr, "Invalid metadata key", nil)
		}

		categoryValidators, ok := validators[category[0]]
		if !ok {
			return validated, serializer.NewError(serializer.CodeParamErr, "Invalid metadata key",
				fmt.Errorf("unknown category: %s", category[0]))
		}

		// Explicit validators
		if v, ok := categoryValidators[patch.Key]; ok {
			if err := v(ctx, m, &patch); err != nil {
				return validated, serializer.NewError(serializer.CodeParamErr, "Invalid metadata patch", err)
			}
		}

		// Wildcard validators
		if v, ok := categoryValidators[wildcardMetadataKey]; ok {
			if err := v(ctx, m, &patch); err != nil {
				return validated, serializer.NewError(serializer.CodeParamErr, "Invalid metadata patch", err)
			}
		}

		validated = append(validated, patch)
	}

	return validated, nil
}
