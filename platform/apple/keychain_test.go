//go:build darwin

package apple

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestKeychain_ReadDistinguishesAbsenceAndCancellationFromErrors(t *testing.T) {
	if found, err := readStatus(errSecSuccess); err != nil || !found {
		t.Fatalf("readStatus(errSecSuccess) = %v, %v", found, err)
	}
	if found, err := readStatus(errSecItemNotFound); err != nil || found {
		t.Fatalf("readStatus(errSecItemNotFound) = %v, %v", found, err)
	}
	if _, err := readStatus(errSecUserCanceled); err != (KeychainErrorUserCanceled{Operation: KeychainOperationRead}) {
		t.Fatalf("readStatus(errSecUserCanceled) = %v", err)
	}
	if _, err := readStatus(errSecAuthFailed); err != (KeychainErrorSecurity{Operation: KeychainOperationRead, Status: errSecAuthFailed}) {
		t.Fatalf("readStatus(errSecAuthFailed) = %v", err)
	}
}

func TestKeychain_PlatformErrorKeepsNativeOperationAndStatus(t *testing.T) {
	err := fmt.Errorf("platform: %w", checkStatus(KeychainOperationUpdate, errSecAuthFailed))
	var keychainErr KeychainError
	if !errors.As(err, &keychainErr) {
		t.Fatalf("errors.As(%v) found no KeychainError", err)
	}
	if keychainErr != (KeychainErrorSecurity{Operation: KeychainOperationUpdate, Status: errSecAuthFailed}) {
		t.Fatalf("KeychainError = %#v", keychainErr)
	}
	if got := keychainErr.Error(); got != "keychain Update failed with OSStatus -25293" {
		t.Fatalf("message = %q", got)
	}
	if got := (KeychainErrorUserCanceled{Operation: KeychainOperationRead}).Error(); got != "Keychain access was cancelled." {
		t.Fatalf("message = %q", got)
	}
	if got := (KeychainErrorInvalidResponse{Message: "keychain returned no item"}).Error(); got != "invalid keychain response: keychain returned no item" {
		t.Fatalf("message = %q", got)
	}
}

func TestKeychain_DeletePreservesMacosMissingItemError(t *testing.T) {
	if err := checkStatus(KeychainOperationDelete, errSecSuccess); err != nil {
		t.Fatalf("checkStatus(errSecSuccess) = %v", err)
	}
	for _, status := range []int32{errSecItemNotFound, errSecAuthFailed} {
		if err := checkStatus(KeychainOperationDelete, status); err != (KeychainErrorSecurity{Operation: KeychainOperationDelete, Status: status}) {
			t.Fatalf("checkStatus(%d) = %v", status, err)
		}
	}
	if err := checkStatus(KeychainOperationDelete, errSecUserCanceled); err != (KeychainErrorUserCanceled{Operation: KeychainOperationDelete}) {
		t.Fatalf("checkStatus(errSecUserCanceled) = %v", err)
	}
}

func TestKeychain_CreateReportsExistingItemsWithoutError(t *testing.T) {
	if created, err := createStatus(errSecSuccess); err != nil || !created {
		t.Fatalf("createStatus(errSecSuccess) = %v, %v", created, err)
	}
	if created, err := createStatus(errSecDuplicateItem); err != nil || created {
		t.Fatalf("createStatus(errSecDuplicateItem) = %v, %v", created, err)
	}
	if _, err := createStatus(errSecItemNotFound); err != (KeychainErrorSecurity{Operation: KeychainOperationCreate, Status: errSecItemNotFound}) {
		t.Fatalf("createStatus(errSecItemNotFound) = %v", err)
	}
}

func TestKeychain_Symbols(t *testing.T) {
	if err := loadSecurity(); err != nil {
		t.Fatal(err)
	}
	for name, ref := range map[string]uintptr{
		"kSecClass":                 uintptr(kSecClass),
		"kSecAttrServer":            uintptr(kSecAttrServer),
		"kSecAttrAccount":           uintptr(kSecAttrAccount),
		"kSecValueData":             uintptr(kSecValueData),
		"kSecClassInternetPassword": uintptr(kSecClassInternetPassword),
		"kSecReturnAttributes":      uintptr(kSecReturnAttributes),
		"kSecReturnData":            uintptr(kSecReturnData),
	} {
		if ref == 0 {
			t.Errorf("%s not resolved", name)
		}
	}
}

func TestKeychain_RoundTrip(t *testing.T) {
	server := fmt.Sprintf("gpui-go-keychain-test-%d-%d.invalid", os.Getpid(), time.Now().UnixNano())
	created, err := WriteCredentialsIfNotExists(server, "alice", []byte("first"))
	if err != nil {
		t.Skipf("Keychain unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = DeleteCredentials(server)
	})
	if !created {
		t.Fatal("fresh item reported as existing")
	}
	if created, err := WriteCredentialsIfNotExists(server, "alice", []byte("second")); err != nil || created {
		t.Fatalf("duplicate add = %v, %v", created, err)
	}
	account, password, ok, err := ReadCredentials(server)
	if err != nil || !ok || account != "alice" || string(password) != "first" {
		t.Fatalf("ReadCredentials = %q, %q, %v, %v", account, password, ok, err)
	}
	if err := WriteCredentials(server, "bob", []byte("third")); err != nil {
		t.Fatal(err)
	}
	account, password, ok, err = ReadCredentials(server)
	if err != nil || !ok || account != "bob" || string(password) != "third" {
		t.Fatalf("ReadCredentials after update = %q, %q, %v, %v", account, password, ok, err)
	}
	if err := DeleteCredentials(server); err != nil {
		t.Fatal(err)
	}
	if _, _, ok, err := ReadCredentials(server); err != nil || ok {
		t.Fatalf("ReadCredentials after delete = %v, %v", ok, err)
	}
	if err := DeleteCredentials(server); err != (KeychainErrorSecurity{Operation: KeychainOperationDelete, Status: errSecItemNotFound}) {
		t.Fatalf("second delete = %v", err)
	}
}

func TestKeychain_InvalidUTF8IsAnError(t *testing.T) {
	if err := loadSecurity(); err != nil {
		t.Skip(err)
	}
	var invalid KeychainErrorInvalidResponse
	if err := WriteCredentials("bad\xff.invalid", "alice", []byte("x")); !errors.As(err, &invalid) {
		t.Errorf("WriteCredentials with an invalid server = %v", err)
	}
	if _, err := WriteCredentialsIfNotExists("gpui-go.invalid", "bad\xff", []byte("x")); !errors.As(err, &invalid) {
		t.Errorf("WriteCredentialsIfNotExists with an invalid account = %v", err)
	}
	if _, _, _, err := ReadCredentials("bad\xff.invalid"); !errors.As(err, &invalid) {
		t.Errorf("ReadCredentials with an invalid server = %v", err)
	}
	if err := DeleteCredentials("bad\xff.invalid"); !errors.As(err, &invalid) {
		t.Errorf("DeleteCredentials with an invalid server = %v", err)
	}
}
