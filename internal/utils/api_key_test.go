package utils

import "testing"

func TestGenerateAndVerifyAPIKeyToken(t *testing.T) {
	originalPepper := DefaultAPIKeyPepper
	DefaultAPIKeyPepper = "test-pepper"
	defer func() {
		DefaultAPIKeyPepper = originalPepper
	}()

	prefix, fullKey, secretHash, err := GenerateAPIKeyToken()
	if err != nil {
		t.Fatalf("GenerateAPIKeyToken returned error: %v", err)
	}
	if prefix == "" || len(prefix) != 15 || fullKey == "" || secretHash == "" {
		t.Fatalf("unexpected token parts: prefix=%q key=%q hash=%q", prefix, fullKey, secretHash)
	}

	gotPrefix, secret, err := SplitAPIKeyToken(fullKey)
	if err != nil {
		t.Fatalf("SplitAPIKeyToken returned error: %v", err)
	}
	if gotPrefix != prefix {
		t.Fatalf("prefix mismatch: got %q want %q", gotPrefix, prefix)
	}
	if !VerifyAPIKeySecret(secret, secretHash) {
		t.Fatalf("VerifyAPIKeySecret should accept generated secret")
	}
	if VerifyAPIKeySecret(secret+"x", secretHash) {
		t.Fatalf("VerifyAPIKeySecret should reject wrong secret")
	}
}

func TestSplitAPIKeyTokenRejectsInvalidFormat(t *testing.T) {
	originalPepper := DefaultAPIKeyPepper
	DefaultAPIKeyPepper = "test-pepper"
	defer func() {
		DefaultAPIKeyPepper = originalPepper
	}()

	_, _, validHash, err := GenerateAPIKeyToken()
	if err != nil {
		t.Fatalf("GenerateAPIKeyToken returned error: %v", err)
	}
	if validHash == "" {
		t.Fatalf("expected non-empty hash")
	}

	cases := []string{
		"",
		"abc",
		"sk_ABC.invalid",
		"sk_short.invalid",
		"sk_abcdefghijkl.invalid+",
		"sk_abcdefghijkl.",
	}
	for _, tc := range cases {
		if _, _, err := SplitAPIKeyToken(tc); err == nil {
			t.Fatalf("SplitAPIKeyToken(%q) expected error", tc)
		}
	}
}
