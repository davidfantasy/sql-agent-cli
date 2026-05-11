package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	t.Setenv(MasterKeyEnv, "0123456789abcdef0123456789abcdef")

	plaintext := []byte(`{"name":"local"}`)
	ciphertext, err := Encrypt(plaintext)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("expected %s, got %s", plaintext, decrypted)
	}
}

func TestEncrypt_AutoGeneratesKey(t *testing.T) {
	t.Setenv(MasterKeyEnv, "")
	t.Setenv("HOME", t.TempDir())

	plaintext := []byte("secret")
	ciphertext, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("expected auto-generated key to succeed, got: %v", err)
	}

	decrypted, err := Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("expected %s, got %s", plaintext, decrypted)
	}
}

func TestSaveLoadDeleteConnection_RoundTrip(t *testing.T) {
	t.Setenv(MasterKeyEnv, "0123456789abcdef0123456789abcdef")
	t.Setenv("HOME", t.TempDir())

	conn := Connection{
		Name:             "local",
		Driver:           "sqlite",
		Database:         "./dev.db",
		CredentialHelper: "helper --profile local",
	}

	if err := SaveConnection(conn); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(storePath(conn.Name))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), conn.Database) {
		t.Fatal("expected stored file to be encrypted, found plaintext database path")
	}

	loaded, err := LoadConnection(conn.Name)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != conn {
		t.Fatalf("expected %+v, got %+v", conn, loaded)
	}

	if err := DeleteConnection(conn.Name); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Clean(storePath(conn.Name))); !os.IsNotExist(err) {
		t.Fatalf("expected connection file to be removed, got %v", err)
	}
}
