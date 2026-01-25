/*
 * Copyright (c) 2026, Psiphon Inc.
 * All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package crypto

import (
	"bytes"
	"crypto/ed25519"
	"strings"
	"testing"

	"github.com/tyler-smith/go-bip39"
)

// TestGenerateMnemonic tests that GenerateMnemonic produces valid BIP-39 mnemonics
func TestGenerateMnemonic(t *testing.T) {
	mnemonic, err := GenerateMnemonic()
	if err != nil {
		t.Fatalf("GenerateMnemonic failed: %v", err)
	}

	// Check that mnemonic is not empty
	if mnemonic == "" {
		t.Fatal("GenerateMnemonic returned empty string")
	}

	// Check that mnemonic is valid according to BIP-39
	if !bip39.IsMnemonicValid(mnemonic) {
		t.Fatalf("GenerateMnemonic returned invalid mnemonic: %s", mnemonic)
	}

	// Check that mnemonic has 24 words (256-bit entropy)
	words := strings.Fields(mnemonic)
	if len(words) != 24 {
		t.Fatalf("GenerateMnemonic returned %d words, expected 24", len(words))
	}
}

// TestDeriveKeyPairFromMnemonic tests that DeriveKeyPairFromMnemonic produces valid Ed25519 keys
func TestDeriveKeyPairFromMnemonic(t *testing.T) {
	// Use a known valid mnemonic for deterministic testing
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art"

	keyPair, err := DeriveKeyPairFromMnemonic(mnemonic, "")
	if err != nil {
		t.Fatalf("DeriveKeyPairFromMnemonic failed: %v", err)
	}

	// Validate key lengths
	if len(keyPair.PrivateKey) != ed25519.PrivateKeySize {
		t.Fatalf("PrivateKey has wrong length: expected %d, got %d", ed25519.PrivateKeySize, len(keyPair.PrivateKey))
	}

	if len(keyPair.PublicKey) != ed25519.PublicKeySize {
		t.Fatalf("PublicKey has wrong length: expected %d, got %d", ed25519.PublicKeySize, len(keyPair.PublicKey))
	}

	// Verify that the public key matches the private key
	derivedPublicKey := ed25519.PrivateKey(keyPair.PrivateKey).Public().(ed25519.PublicKey)
	if !ed25519.PublicKey(keyPair.PublicKey).Equal(derivedPublicKey) {
		t.Fatal("PublicKey does not match the PrivateKey")
	}
}

// TestDeriveKeyPairFromMnemonicInvalid tests that DeriveKeyPairFromMnemonic rejects invalid mnemonics
func TestDeriveKeyPairFromMnemonicInvalid(t *testing.T) {
	invalidMnemonic := "this is not a valid mnemonic phrase"

	_, err := DeriveKeyPairFromMnemonic(invalidMnemonic, "")
	if err == nil {
		t.Fatal("DeriveKeyPairFromMnemonic should reject invalid mnemonic")
	}
}

// TestGenerateMnemonicToDeriveKeyPair tests the full flow: GenerateMnemonic -> DeriveKeyPairFromMnemonic
// This is the specific test requested by tmgrask
func TestGenerateMnemonicToDeriveKeyPair(t *testing.T) {
	// Generate a new mnemonic
	mnemonic, err := GenerateMnemonic()
	if err != nil {
		t.Fatalf("GenerateMnemonic failed: %v", err)
	}

	// Derive key pair from the generated mnemonic
	keyPair, err := DeriveKeyPairFromMnemonic(mnemonic, "")
	if err != nil {
		t.Fatalf("DeriveKeyPairFromMnemonic failed with generated mnemonic: %v", err)
	}

	// Validate that the derived key pair is valid
	if len(keyPair.PrivateKey) != ed25519.PrivateKeySize {
		t.Fatalf("PrivateKey has wrong length: expected %d, got %d", ed25519.PrivateKeySize, len(keyPair.PrivateKey))
	}

	if len(keyPair.PublicKey) != ed25519.PublicKeySize {
		t.Fatalf("PublicKey has wrong length: expected %d, got %d", ed25519.PublicKeySize, len(keyPair.PublicKey))
	}

	// Verify that the public key matches the private key
	derivedPublicKey := ed25519.PrivateKey(keyPair.PrivateKey).Public().(ed25519.PublicKey)
	if !ed25519.PublicKey(keyPair.PublicKey).Equal(derivedPublicKey) {
		t.Fatal("PublicKey does not match the PrivateKey")
	}

	// Test that we can sign and verify with the generated keys
	message := []byte("test message")
	signature := ed25519.Sign(keyPair.PrivateKey, message)
	if !ed25519.Verify(keyPair.PublicKey, message, signature) {
		t.Fatal("Failed to verify signature with derived keys")
	}
}

// TestGenerateKeyPair tests that GenerateKeyPair produces valid Ed25519 keys
func TestGenerateKeyPair(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Validate key lengths
	if len(keyPair.PrivateKey) != ed25519.PrivateKeySize {
		t.Fatalf("PrivateKey has wrong length: expected %d, got %d", ed25519.PrivateKeySize, len(keyPair.PrivateKey))
	}

	if len(keyPair.PublicKey) != ed25519.PublicKeySize {
		t.Fatalf("PublicKey has wrong length: expected %d, got %d", ed25519.PublicKeySize, len(keyPair.PublicKey))
	}

	// Verify that the public key matches the private key
	derivedPublicKey := ed25519.PrivateKey(keyPair.PrivateKey).Public().(ed25519.PublicKey)
	if !ed25519.PublicKey(keyPair.PublicKey).Equal(derivedPublicKey) {
		t.Fatal("PublicKey does not match the PrivateKey")
	}

	// Test that we can sign and verify with the generated keys
	message := []byte("test message")
	signature := ed25519.Sign(keyPair.PrivateKey, message)
	if !ed25519.Verify(keyPair.PublicKey, message, signature) {
		t.Fatal("Failed to verify signature with generated keys")
	}
}

// TestParsePrivateKey tests that ParsePrivateKey correctly reconstructs a KeyPair
func TestParsePrivateKey(t *testing.T) {
	// Generate a key pair first
	original, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Parse the private key
	parsed, err := ParsePrivateKey(original.PrivateKey)
	if err != nil {
		t.Fatalf("ParsePrivateKey failed: %v", err)
	}

	// Verify that the parsed key pair matches the original
	if !bytes.Equal(parsed.PrivateKey, original.PrivateKey) {
		t.Fatal("Parsed PrivateKey does not match original")
	}

	if !bytes.Equal(parsed.PublicKey, original.PublicKey) {
		t.Fatal("Parsed PublicKey does not match original")
	}
}

// TestParsePrivateKeyInvalidLength tests that ParsePrivateKey rejects invalid key lengths
func TestParsePrivateKeyInvalidLength(t *testing.T) {
	invalidKey := make([]byte, 32) // Too short

	_, err := ParsePrivateKey(invalidKey)
	if err == nil {
		t.Fatal("ParsePrivateKey should reject keys with invalid length")
	}
}

// TestDeriveKeyPairWithPath tests that different paths produce different keys
func TestDeriveKeyPairWithPath(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art"

	keyPair1, err := DeriveKeyPairFromMnemonic(mnemonic, "")
	if err != nil {
		t.Fatalf("DeriveKeyPairFromMnemonic failed: %v", err)
	}

	keyPair2, err := DeriveKeyPairFromMnemonic(mnemonic, "/path1")
	if err != nil {
		t.Fatalf("DeriveKeyPairFromMnemonic failed: %v", err)
	}

	// Verify that different paths produce different keys
	if ed25519.PublicKey(keyPair1.PublicKey).Equal(keyPair2.PublicKey) {
		t.Fatal("Different paths should produce different keys")
	}
}

// TestDeterministicDerivation tests that the same mnemonic always produces the same keys
func TestDeterministicDerivation(t *testing.T) {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art"

	keyPair1, err := DeriveKeyPairFromMnemonic(mnemonic, "")
	if err != nil {
		t.Fatalf("DeriveKeyPairFromMnemonic failed: %v", err)
	}

	keyPair2, err := DeriveKeyPairFromMnemonic(mnemonic, "")
	if err != nil {
		t.Fatalf("DeriveKeyPairFromMnemonic failed: %v", err)
	}

	// Verify that the same mnemonic produces the same keys
	if !bytes.Equal(keyPair1.PublicKey, keyPair2.PublicKey) {
		t.Fatal("Same mnemonic should produce identical keys")
	}

	if !bytes.Equal(keyPair1.PrivateKey, keyPair2.PrivateKey) {
		t.Fatal("Same mnemonic should produce identical private keys")
	}
}
