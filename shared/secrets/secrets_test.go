package secrets

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/devboxos/devboxos/shared/types"
)

func TestEnvProvider_Resolve(t *testing.T) {
	provider := NewEnvProvider()

	t.Run("existing env var", func(t *testing.T) {
		os.Setenv("TEST_SECRET_VAR", "test-value-123")
		defer os.Unsetenv("TEST_SECRET_VAR")

		value, err := provider.Resolve("TEST_SECRET_VAR")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if value != "test-value-123" {
			t.Fatalf("expected test-value-123, got %s", value)
		}
	})

	t.Run("missing env var", func(t *testing.T) {
		_, err := provider.Resolve("NONEXISTENT_VAR_12345")
		if err == nil {
			t.Fatal("expected error for missing env var")
		}
	})

	t.Run("empty source", func(t *testing.T) {
		_, err := provider.Resolve("")
		if err == nil {
			t.Fatal("expected error for empty source")
		}
	})
}

func TestGenerateProvider_Resolve(t *testing.T) {
	provider := NewGenerateProvider()

	t.Run("default length", func(t *testing.T) {
		value, err := provider.Resolve("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(value) != defaultSecretLength {
			t.Fatalf("expected length %d, got %d", defaultSecretLength, len(value))
		}
	})

	t.Run("specific length", func(t *testing.T) {
		value, err := provider.Resolve("16")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(value) != 16 {
			t.Fatalf("expected length 16, got %d", len(value))
		}
	})

	t.Run("hex charset", func(t *testing.T) {
		value, err := provider.Resolve("32:hex")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(value) != 32 {
			t.Fatalf("expected length 32, got %d", len(value))
		}
		for _, c := range value {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Fatalf("invalid hex character: %c", c)
			}
		}
	})

	t.Run("alphanumeric charset", func(t *testing.T) {
		value, err := provider.Resolve("20:alphanumeric")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(value) != 20 {
			t.Fatalf("expected length 20, got %d", len(value))
		}
		for _, c := range value {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				t.Fatalf("invalid alphanumeric character: %c", c)
			}
		}
	})

	t.Run("too short", func(t *testing.T) {
		_, err := provider.Resolve("4")
		if err == nil {
			t.Fatal("expected error for too short secret")
		}
	})

	t.Run("too long", func(t *testing.T) {
		_, err := provider.Resolve("300")
		if err == nil {
			t.Fatal("expected error for too long secret")
		}
	})

	t.Run("unsupported charset", func(t *testing.T) {
		_, err := provider.Resolve("32:binary")
		if err == nil {
			t.Fatal("expected error for unsupported charset")
		}
	})
}

func TestFileProvider_Resolve(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("existing file", func(t *testing.T) {
		secretFile := filepath.Join(tempDir, "secret.txt")
		if err := os.WriteFile(secretFile, []byte("file-secret-value\n"), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		provider := NewFileProvider(tempDir)
		value, err := provider.Resolve("secret.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if value != "file-secret-value" {
			t.Fatalf("expected file-secret-value, got %s", value)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		provider := NewFileProvider(tempDir)
		_, err := provider.Resolve("nonexistent.txt")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		secretFile := filepath.Join(tempDir, "empty.txt")
		if err := os.WriteFile(secretFile, []byte(""), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		provider := NewFileProvider(tempDir)
		_, err := provider.Resolve("empty.txt")
		if err == nil {
			t.Fatal("expected error for empty file")
		}
	})

	t.Run("directory instead of file", func(t *testing.T) {
		if err := os.MkdirAll(filepath.Join(tempDir, "dir"), 0755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		provider := NewFileProvider(tempDir)
		_, err := provider.Resolve("dir")
		if err == nil {
			t.Fatal("expected error for directory")
		}
	})
}

func TestRegistry_Resolve(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewEnvProvider())
	registry.Register(NewGenerateProvider())

	t.Run("env provider", func(t *testing.T) {
		os.Setenv("REGISTRY_TEST_VAR", "registry-value")
		defer os.Unsetenv("REGISTRY_TEST_VAR")

		value, err := registry.Resolve("env:REGISTRY_TEST_VAR")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if value != "registry-value" {
			t.Fatalf("expected registry-value, got %s", value)
		}
	})

	t.Run("generate provider", func(t *testing.T) {
		value, err := registry.Resolve("generate:16")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(value) != 16 {
			t.Fatalf("expected length 16, got %d", len(value))
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		_, err := registry.Resolve("unknown:args")
		if err == nil {
			t.Fatal("expected error for unknown provider")
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		_, err := registry.Resolve("no-colon-here")
		if err == nil {
			t.Fatal("expected error for invalid format")
		}
	})
}

func TestAgeCrypto_EncryptDecrypt(t *testing.T) {
	crypto, err := NewAgeCrypto()
	if err != nil {
		t.Fatalf("failed to create crypto: %v", err)
	}

	plaintext := []byte("this is a secret message")

	ciphertext, err := crypto.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Fatal("ciphertext is empty")
	}

	decrypted, err := crypto.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("expected %s, got %s", plaintext, decrypted)
	}
}

func TestAgeCrypto_LoadOrCreateKey(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "key.txt")

	t.Run("create new key", func(t *testing.T) {
		crypto, err := LoadOrCreateKey(keyPath)
		if err != nil {
			t.Fatalf("failed to load or create key: %v", err)
		}
		if crypto == nil {
			t.Fatal("crypto is nil")
		}

		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			t.Fatal("key file was not created")
		}
	})

	t.Run("load existing key", func(t *testing.T) {
		crypto1, err := LoadOrCreateKey(keyPath)
		if err != nil {
			t.Fatalf("failed to load or create key: %v", err)
		}

		crypto2, err := LoadOrCreateKey(keyPath)
		if err != nil {
			t.Fatalf("failed to load or create key: %v", err)
		}

		if crypto1.IdentityString() != crypto2.IdentityString() {
			t.Fatal("loaded key does not match created key")
		}
	})
}

func TestStore_SetGetListDelete(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "key.txt")
	storePath := filepath.Join(tempDir, "secrets.enc")

	crypto, err := LoadOrCreateKey(keyPath)
	if err != nil {
		t.Fatalf("failed to load or create key: %v", err)
	}

	store := NewStore(crypto, storePath)

	t.Run("set and get", func(t *testing.T) {
		if err := store.Set("TEST_SECRET", "test-value", "env:TEST_VAR"); err != nil {
			t.Fatalf("failed to set secret: %v", err)
		}

		entry, err := store.Get("TEST_SECRET")
		if err != nil {
			t.Fatalf("failed to get secret: %v", err)
		}
		if entry.Value != "test-value" {
			t.Fatalf("expected test-value, got %s", entry.Value)
		}
		if entry.Source != "env:TEST_VAR" {
			t.Fatalf("expected env:TEST_VAR, got %s", entry.Source)
		}
	})

	t.Run("list masks values", func(t *testing.T) {
		entries := store.List()
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Value != "****" {
			t.Fatalf("expected masked value, got %s", entries[0].Value)
		}
	})

	t.Run("delete", func(t *testing.T) {
		if err := store.Delete("TEST_SECRET"); err != nil {
			t.Fatalf("failed to delete secret: %v", err)
		}

		_, err := store.Get("TEST_SECRET")
		if err == nil {
			t.Fatal("expected error after delete")
		}
	})

	t.Run("get nonexistent", func(t *testing.T) {
		_, err := store.Get("NONEXISTENT")
		if err == nil {
			t.Fatal("expected error for nonexistent secret")
		}
	})
}

func TestStore_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "key.txt")
	storePath := filepath.Join(tempDir, "secrets.enc")

	crypto, err := LoadOrCreateKey(keyPath)
	if err != nil {
		t.Fatalf("failed to load or create key: %v", err)
	}

	store1 := NewStore(crypto, storePath)
	if err := store1.Set("PERSISTENT_SECRET", "persistent-value", "generate:32"); err != nil {
		t.Fatalf("failed to set secret: %v", err)
	}

	store2 := NewStore(crypto, storePath)
	if err := store2.Load(); err != nil {
		t.Fatalf("failed to load store: %v", err)
	}

	entry, err := store2.Get("PERSISTENT_SECRET")
	if err != nil {
		t.Fatalf("failed to get secret: %v", err)
	}
	if entry.Value != "persistent-value" {
		t.Fatalf("expected persistent-value, got %s", entry.Value)
	}
}

func TestResolver_Resolve(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "key.txt")
	storePath := filepath.Join(tempDir, "secrets.enc")

	os.Setenv("RESOLVER_TEST_VAR", "resolver-test-value")
	defer os.Unsetenv("RESOLVER_TEST_VAR")

	resolver, err := NewResolver(tempDir, keyPath, storePath)
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	t.Run("resolve from env", func(t *testing.T) {
		ref := types.SecretRef{Name: "TEST_ENV_SECRET", Source: "env:RESOLVER_TEST_VAR"}
		value, err := resolver.Resolve(ref)
		if err != nil {
			t.Fatalf("failed to resolve: %v", err)
		}
		if value != "resolver-test-value" {
			t.Fatalf("expected resolver-test-value, got %s", value)
		}
	})

	t.Run("resolve from generate", func(t *testing.T) {
		ref := types.SecretRef{Name: "TEST_GEN_SECRET", Source: "generate:32"}
		value, err := resolver.Resolve(ref)
		if err != nil {
			t.Fatalf("failed to resolve: %v", err)
		}
		if len(value) != 32 {
			t.Fatalf("expected length 32, got %d", len(value))
		}
	})

	t.Run("resolve from file", func(t *testing.T) {
		secretFile := filepath.Join(tempDir, "file-secret.txt")
		if err := os.WriteFile(secretFile, []byte("file-secret-content"), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		ref := types.SecretRef{Name: "TEST_FILE_SECRET", Source: "file:file-secret.txt"}
		value, err := resolver.Resolve(ref)
		if err != nil {
			t.Fatalf("failed to resolve: %v", err)
		}
		if value != "file-secret-content" {
			t.Fatalf("expected file-secret-content, got %s", value)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		ref := types.SecretRef{Name: "", Source: "generate:32"}
		_, err := resolver.Resolve(ref)
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("empty source", func(t *testing.T) {
		ref := types.SecretRef{Name: "TEST", Source: ""}
		_, err := resolver.Resolve(ref)
		if err == nil {
			t.Fatal("expected error for empty source")
		}
	})
}

func TestResolver_SetGetListDelete(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "key.txt")
	storePath := filepath.Join(tempDir, "secrets.enc")

	resolver, err := NewResolver(tempDir, keyPath, storePath)
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	t.Run("set", func(t *testing.T) {
		if err := resolver.Set("MANUAL_SECRET", "manual-value"); err != nil {
			t.Fatalf("failed to set: %v", err)
		}
	})

	t.Run("get", func(t *testing.T) {
		value, err := resolver.Get("MANUAL_SECRET")
		if err != nil {
			t.Fatalf("failed to get: %v", err)
		}
		if value != "manual-value" {
			t.Fatalf("expected manual-value, got %s", value)
		}
	})

	t.Run("list", func(t *testing.T) {
		entries := resolver.List()
		if len(entries) < 1 {
			t.Fatalf("expected at least 1 entry, got %d", len(entries))
		}
	})

	t.Run("delete", func(t *testing.T) {
		if err := resolver.Delete("MANUAL_SECRET"); err != nil {
			t.Fatalf("failed to delete: %v", err)
		}

		_, err := resolver.Get("MANUAL_SECRET")
		if err == nil {
			t.Fatal("expected error after delete")
		}
	})
}

func TestResolver_Rotate(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "key.txt")
	storePath := filepath.Join(tempDir, "secrets.enc")

	resolver, err := NewResolver(tempDir, keyPath, storePath)
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	t.Run("rotate generated secret", func(t *testing.T) {
		ref := types.SecretRef{Name: "ROTATABLE_SECRET", Source: "generate:32"}
		value1, err := resolver.Resolve(ref)
		if err != nil {
			t.Fatalf("failed to resolve: %v", err)
		}

		if err := resolver.Rotate("ROTATABLE_SECRET"); err != nil {
			t.Fatalf("failed to rotate: %v", err)
		}

		value2, err := resolver.Get("ROTATABLE_SECRET")
		if err != nil {
			t.Fatalf("failed to get after rotate: %v", err)
		}

		if value1 == value2 {
			t.Fatal("secret was not rotated (values are the same)")
		}
	})

	t.Run("cannot rotate manual secret", func(t *testing.T) {
		if err := resolver.Set("MANUAL_ROTATE_TEST", "value"); err != nil {
			t.Fatalf("failed to set: %v", err)
		}

		err := resolver.Rotate("MANUAL_ROTATE_TEST")
		if err == nil {
			t.Fatal("expected error when rotating manual secret")
		}
	})
}
