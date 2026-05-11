package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
)

const MasterKeyEnv = "DB_AGENT_MASTER_KEY"

func masterKeyPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".sql-agent", ".master_key")
}

func loadKey() ([]byte, error) {
	// 1. Try environment variable first
	key := os.Getenv(MasterKeyEnv)
	if key != "" {
		hash := sha256.Sum256([]byte(key))
		return hash[:], nil
	}

	// 2. Try reading from file
	path := masterKeyPath()
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		hash := sha256.Sum256(data)
		return hash[:], nil
	}

	// 3. Generate a new key and save it
	newKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, newKey, 0o600); err != nil {
		return nil, err
	}

	hash := sha256.Sum256(newKey)
	return hash[:], nil
}

func Encrypt(plaintext []byte) ([]byte, error) {
	key, err := loadKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func Decrypt(ciphertext []byte) ([]byte, error) {
	key, err := loadKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, data := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, data, nil)
}
