package cmd

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/cloudreve/Cloudreve/v4/application/dependency"
	"github.com/cloudreve/Cloudreve/v4/ent/entity"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/encrypt"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/spf13/cobra"
)

var (
	outputToFile     string
	newMasterKeyFile string
)

func init() {
	rootCmd.AddCommand(masterKeyCmd)
	masterKeyCmd.AddCommand(masterKeyGenerateCmd)
	masterKeyCmd.AddCommand(masterKeyGetCmd)
	masterKeyCmd.AddCommand(masterKeyRotateCmd)

	masterKeyGenerateCmd.Flags().StringVarP(&outputToFile, "output", "o", "", "Output master key to file instead of stdout")
	masterKeyRotateCmd.Flags().StringVarP(&newMasterKeyFile, "new-key", "n", "", "Path to file containing the new master key (base64 encoded).")
}

var masterKeyCmd = &cobra.Command{
	Use:   "master-key",
	Short: "Master encryption key management",
	Long:  "Manage master encryption keys for file encryption. Use subcommands to generate, get, or rotate keys.",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var masterKeyGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new master encryption key",
	Long:  "Generate a new random 32-byte (256-bit) master encryption key and output it in base64 format.",
	Run: func(cmd *cobra.Command, args []string) {
		// Generate 32-byte random key
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to generate random key: %v\n", err)
			os.Exit(1)
		}

		// Encode to base64
		encodedKey := base64.StdEncoding.EncodeToString(key)

		if outputToFile != "" {
			// Write to file
			if err := os.WriteFile(outputToFile, []byte(encodedKey), 0600); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to write key to file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Master key generated and saved to: %s\n", outputToFile)
		} else {
			// Output to stdout
			fmt.Println(encodedKey)
		}
	},
}

var masterKeyGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the current master encryption key",
	Long:  "Retrieve and display the current master encryption key from the configured vault (setting, env, or file).",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		dep := dependency.NewDependency(
			dependency.WithConfigPath(confPath),
		)
		logger := dep.Logger()

		// Get the master key vault
		vault := encrypt.NewMasterEncryptKeyVault(ctx, dep.SettingProvider())

		// Retrieve the master key
		key, err := vault.GetMasterKey(ctx)
		if err != nil {
			logger.Error("Failed to get master key: %s", err)
			os.Exit(1)
		}

		// Encode to base64 and display
		encodedKey := base64.StdEncoding.EncodeToString(key)
		fmt.Println("")
		fmt.Println(encodedKey)
	},
}

var masterKeyRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate the master encryption key",
	Long: `Rotate the master encryption key by re-encrypting all encrypted file keys with a new master key.
This operation:
1. Retrieves the current master key
2. Loads a new master key from file
3. Re-encrypts all file encryption keys with the new master key
4. Updates the master key in the settings database

Warning: This is a critical operation. Make sure to backup your database before proceeding.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		dep := dependency.NewDependency(
			dependency.WithConfigPath(confPath),
		)
		logger := dep.Logger()

		logger.Info("Starting master key rotation...")

		// Get the old master key
		vault := encrypt.NewMasterEncryptKeyVault(ctx, dep.SettingProvider())
		oldMasterKey, err := vault.GetMasterKey(ctx)
		if err != nil {
			logger.Error("Failed to get current master key: %s", err)
			os.Exit(1)
		}
		logger.Info("Retrieved current master key")

		// Get or generate the new master key
		var newMasterKey []byte
		// Load from file
		keyData, err := os.ReadFile(newMasterKeyFile)
		if err != nil {
			logger.Error("Failed to read new master key file: %s", err)
			os.Exit(1)
		}
		newMasterKey, err = base64.StdEncoding.DecodeString(string(keyData))
		if err != nil {
			logger.Error("Failed to decode new master key: %s", err)
			os.Exit(1)
		}
		if len(newMasterKey) != 32 {
			logger.Error("Invalid new master key: must be 32 bytes (256 bits), got %d bytes", len(newMasterKey))
			os.Exit(1)
		}
		logger.Info("Loaded new master key from file: %s", newMasterKeyFile)

		// Query all entities with encryption metadata
		db := dep.DBClient()
		entities, err := db.Entity.Query().
			Where(entity.Not(entity.PropsIsNil())).
			All(ctx)
		if err != nil {
			logger.Error("Failed to query entities: %s", err)
			os.Exit(1)
		}

		logger.Info("Found %d entities to check for encryption", len(entities))

		// Re-encrypt each entity's encryption key
		encryptedCount := 0
		for _, ent := range entities {
			if ent.Props == nil || ent.Props.EncryptMetadata == nil {
				continue
			}

			encMeta := ent.Props.EncryptMetadata

			// Decrypt the file key with old master key
			decryptedFileKey, err := encrypt.DecryptWithMasterKey(oldMasterKey, encMeta.Key)
			if err != nil {
				logger.Error("Failed to decrypt key for entity %d: %s", ent.ID, err)
				os.Exit(1)
			}

			// Re-encrypt the file key with new master key
			newEncryptedKey, err := encrypt.EncryptWithMasterKey(newMasterKey, decryptedFileKey)
			if err != nil {
				logger.Error("Failed to re-encrypt key for entity %d: %s", ent.ID, err)
				os.Exit(1)
			}

			// Update the entity
			newProps := *ent.Props
			newProps.EncryptMetadata = &types.EncryptMetadata{
				Algorithm:    encMeta.Algorithm,
				Key:          newEncryptedKey,
				KeyPlainText: nil, // Don't store plaintext
				IV:           encMeta.IV,
			}

			err = db.Entity.UpdateOne(ent).
				SetProps(&newProps).
				Exec(ctx)
			if err != nil {
				logger.Error("Failed to update entity %d: %s", ent.ID, err)
				os.Exit(1)
			}

			encryptedCount++
		}

		logger.Info("Re-encrypted %d file keys", encryptedCount)

		// Update the master key in settings
		keyStore := dep.SettingProvider().MasterEncryptKeyVault(ctx)
		if keyStore == setting.MasterEncryptKeyVaultTypeSetting {
			encodedNewKey := base64.StdEncoding.EncodeToString(newMasterKey)
			err = dep.SettingClient().Set(ctx, map[string]string{
				"encrypt_master_key": encodedNewKey,
			})
			if err != nil {
				logger.Error("Failed to update master key in settings: %s", err)
				logger.Error("WARNING: File keys have been re-encrypted but master key update failed!")
				logger.Error("Please manually update the encrypt_master_key setting.")
				os.Exit(1)
			}
		} else {
			logger.Info("Current master key is stored in %q", keyStore)
			if keyStore == setting.MasterEncryptKeyVaultTypeEnv {
				logger.Info("Please update the new master encryption key in your \"CR_ENCRYPT_MASTER_KEY\" environment variable.")
			} else if keyStore == setting.MasterEncryptKeyVaultTypeFile {
				logger.Info("Please update the new master encryption key in your key file: %q", dep.SettingProvider().MasterEncryptKeyFile(ctx))
			}
			logger.Info("Last step: Please manually update the new master encryption key in your ENV or key file.")
		}

		logger.Info("Master key rotation completed successfully")
	},
}
