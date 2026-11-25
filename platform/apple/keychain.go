//go:build darwin

package apple

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/megascan/gpui-go/internal/apple/cf"
)

// KeychainOperation is the Keychain call that failed.
type KeychainOperation uint8

const (
	// KeychainOperationRead is SecItemCopyMatching.
	KeychainOperationRead KeychainOperation = iota
	// KeychainOperationUpdate is SecItemUpdate.
	KeychainOperationUpdate
	// KeychainOperationCreate is SecItemAdd.
	KeychainOperationCreate
	// KeychainOperationDelete is SecItemDelete.
	KeychainOperationDelete
)

// String returns the operation's name.
func (o KeychainOperation) String() string {
	switch o {
	case KeychainOperationRead:
		return "Read"
	case KeychainOperationUpdate:
		return "Update"
	case KeychainOperationCreate:
		return "Create"
	case KeychainOperationDelete:
		return "Delete"
	default:
		return fmt.Sprintf("KeychainOperation(%d)", uint8(o))
	}
}

// KeychainError is a Keychain failure: KeychainErrorUserCanceled, KeychainErrorSecurity or
// KeychainErrorInvalidResponse.
type KeychainError interface {
	error
	isKeychainError()
}

// KeychainErrorUserCanceled is returned when the user cancelled Keychain access.
type KeychainErrorUserCanceled struct {
	Operation KeychainOperation
}

// KeychainErrorSecurity is a Security framework failure with its OSStatus.
type KeychainErrorSecurity struct {
	Operation KeychainOperation
	Status    int32
}

// KeychainErrorInvalidResponse is a Keychain reply of an unexpected shape.
type KeychainErrorInvalidResponse struct {
	Message string
}

func (KeychainErrorUserCanceled) isKeychainError()    {}
func (KeychainErrorSecurity) isKeychainError()        {}
func (KeychainErrorInvalidResponse) isKeychainError() {}

// Error returns upstream's message.
func (e KeychainErrorUserCanceled) Error() string {
	return "Keychain access was cancelled."
}

// Error returns upstream's message.
func (e KeychainErrorSecurity) Error() string {
	return fmt.Sprintf("keychain %v failed with OSStatus %d", e.Operation, e.Status)
}

// Error returns upstream's message.
func (e KeychainErrorInvalidResponse) Error() string {
	return "invalid keychain response: " + e.Message
}

const (
	errSecSuccess       int32 = 0
	errSecUserCanceled  int32 = -128
	errSecAuthFailed    int32 = -25293
	errSecDuplicateItem int32 = -25299
	errSecItemNotFound  int32 = -25300
)

const securityPath = "/System/Library/Frameworks/Security.framework/Security"

var (
	securityOnce sync.Once
	securityErr  error

	secItemAdd          func(attributes cf.Dictionary, result *cf.TypeRef) int32
	secItemCopyMatching func(query cf.Dictionary, result *cf.TypeRef) int32
	secItemUpdate       func(query cf.Dictionary, attributes cf.Dictionary) int32
	secItemDelete       func(query cf.Dictionary) int32

	kSecClass                 cf.TypeRef
	kSecAttrServer            cf.TypeRef
	kSecAttrAccount           cf.TypeRef
	kSecValueData             cf.TypeRef
	kSecClassInternetPassword cf.TypeRef
	kSecReturnAttributes      cf.TypeRef
	kSecReturnData            cf.TypeRef
)

func loadSecurity() error {
	securityOnce.Do(func() {
		if err := cf.Load(); err != nil {
			securityErr = err
			return
		}
		lib, err := purego.Dlopen(securityPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			securityErr = fmt.Errorf("apple: open %s: %w", securityPath, err)
			return
		}
		for _, b := range []struct {
			fptr any
			name string
		}{
			{&secItemAdd, "SecItemAdd"},
			{&secItemCopyMatching, "SecItemCopyMatching"},
			{&secItemUpdate, "SecItemUpdate"},
			{&secItemDelete, "SecItemDelete"},
		} {
			addr, err := purego.Dlsym(lib, b.name)
			if err != nil {
				securityErr = fmt.Errorf("apple: %w", err)
				return
			}
			purego.RegisterFunc(b.fptr, addr)
		}
		for _, c := range []struct {
			dst  *cf.TypeRef
			name string
		}{
			{&kSecClass, "kSecClass"},
			{&kSecAttrServer, "kSecAttrServer"},
			{&kSecAttrAccount, "kSecAttrAccount"},
			{&kSecValueData, "kSecValueData"},
			{&kSecClassInternetPassword, "kSecClassInternetPassword"},
			{&kSecReturnAttributes, "kSecReturnAttributes"},
			{&kSecReturnData, "kSecReturnData"},
		} {
			ref, err := cf.Constant(lib, c.name)
			if err != nil {
				securityErr = err
				return
			}
			*c.dst = ref
		}
	})
	return securityErr
}

func securityUnavailable(err error) KeychainError {
	return KeychainErrorInvalidResponse{Message: "Security framework unavailable: " + err.Error()}
}

// WriteCredentials stores password for account on server, updating an existing item for server.
func WriteCredentials(server string, account string, password []byte) error {
	if err := loadSecurity(); err != nil {
		return securityUnavailable(err)
	}
	serverRef, accountRef, passwordRef, err := credentialRefs(server, account, password)
	if err != nil {
		return err
	}
	defer serverRef.Release()
	defer accountRef.Release()
	defer passwordRef.Release()
	query, err := dictionaryRef(
		[]cf.TypeRef{kSecClass, kSecAttrServer},
		[]cf.TypeRef{kSecClassInternetPassword, serverRef.TypeRef()},
	)
	if err != nil {
		return err
	}
	defer query.Release()
	attributes, err := dictionaryRef(
		[]cf.TypeRef{kSecAttrAccount, kSecValueData},
		[]cf.TypeRef{accountRef.TypeRef(), passwordRef.TypeRef()},
	)
	if err != nil {
		return err
	}
	defer attributes.Release()
	operation := KeychainOperationUpdate
	status := secItemUpdate(query.Ref(), attributes.Ref())
	if status == errSecItemNotFound {
		operation = KeychainOperationCreate
		status, err = addCredentials(serverRef.Ref(), accountRef.Ref(), passwordRef.Ref())
		if err != nil {
			return err
		}
	}
	return checkStatus(operation, status)
}

// WriteCredentialsIfNotExists atomically adds a Keychain item unless one with this server and
// account exists. It returns false if the item exists, leaving its secret unchanged. Items for the
// same server under a different account are not conflicts.
func WriteCredentialsIfNotExists(server string, account string, password []byte) (bool, error) {
	if err := loadSecurity(); err != nil {
		return false, securityUnavailable(err)
	}
	serverRef, accountRef, passwordRef, err := credentialRefs(server, account, password)
	if err != nil {
		return false, err
	}
	defer serverRef.Release()
	defer accountRef.Release()
	defer passwordRef.Release()
	status, err := addCredentials(serverRef.Ref(), accountRef.Ref(), passwordRef.Ref())
	if err != nil {
		return false, err
	}
	return createStatus(status)
}

func stringRef(what string, s string) (*cf.Owned[cf.String], error) {
	ref := cf.StringCreate(s)
	if ref == nil {
		return nil, KeychainErrorInvalidResponse{Message: what + " is not valid UTF-8"}
	}
	return ref, nil
}

func credentialRefs(server string, account string, password []byte) (*cf.Owned[cf.String], *cf.Owned[cf.String], *cf.Owned[cf.Data], error) {
	serverRef, err := stringRef("server", server)
	if err != nil {
		return nil, nil, nil, err
	}
	accountRef, err := stringRef("account", account)
	if err != nil {
		serverRef.Release()
		return nil, nil, nil, err
	}
	passwordRef := cf.DataCreate(password)
	if passwordRef == nil {
		serverRef.Release()
		accountRef.Release()
		return nil, nil, nil, KeychainErrorInvalidResponse{Message: "could not create password data"}
	}
	return serverRef, accountRef, passwordRef, nil
}

func dictionaryRef(keys []cf.TypeRef, values []cf.TypeRef) (*cf.Owned[cf.Dictionary], error) {
	ref := cf.DictionaryCreate(keys, values)
	if ref == nil {
		return nil, KeychainErrorInvalidResponse{Message: "could not create keychain query"}
	}
	return ref, nil
}

func addCredentials(server cf.String, account cf.String, password cf.Data) (int32, error) {
	attributes, err := dictionaryRef(
		[]cf.TypeRef{kSecClass, kSecAttrServer, kSecAttrAccount, kSecValueData},
		[]cf.TypeRef{kSecClassInternetPassword, cf.TypeRef(server), cf.TypeRef(account), cf.TypeRef(password)},
	)
	if err != nil {
		return 0, err
	}
	defer attributes.Release()
	return secItemAdd(attributes.Ref(), nil), nil
}

// ReadCredentials reads the account and password stored for server; ok is false when no item
// exists.
func ReadCredentials(server string) (account string, password []byte, ok bool, err error) {
	if err := loadSecurity(); err != nil {
		return "", nil, false, securityUnavailable(err)
	}
	serverRef, err := stringRef("server", server)
	if err != nil {
		return "", nil, false, err
	}
	defer serverRef.Release()
	trueValue := cf.TypeRef(cf.BooleanTrue())
	if trueValue == 0 {
		return "", nil, false, KeychainErrorInvalidResponse{Message: "Core Foundation true value unavailable"}
	}
	query, err := dictionaryRef(
		[]cf.TypeRef{kSecClass, kSecAttrServer, kSecReturnAttributes, kSecReturnData},
		[]cf.TypeRef{kSecClassInternetPassword, serverRef.TypeRef(), trueValue, trueValue},
	)
	if err != nil {
		return "", nil, false, err
	}
	defer query.Release()
	var result cf.TypeRef
	status := secItemCopyMatching(query.Ref(), &result)
	found, err := readStatus(status)
	if err != nil {
		return "", nil, false, err
	}
	if !found {
		return "", nil, false, nil
	}
	if result == 0 {
		return "", nil, false, KeychainErrorInvalidResponse{Message: "keychain returned no item"}
	}
	owned := cf.Own(result)
	defer owned.Release()
	dict, isDict := cf.AsDictionary(owned.Ref())
	if !isDict {
		return "", nil, false, KeychainErrorInvalidResponse{Message: "keychain item was not a dictionary"}
	}
	accountValue, present := dict.Get(kSecAttrAccount)
	if !present {
		return "", nil, false, KeychainErrorInvalidResponse{Message: "account was missing from keychain item"}
	}
	accountString, isString := cf.AsString(accountValue)
	if !isString {
		return "", nil, false, KeychainErrorInvalidResponse{Message: "account was not a string"}
	}
	passwordValue, present := dict.Get(kSecValueData)
	if !present {
		return "", nil, false, KeychainErrorInvalidResponse{Message: "password was missing from keychain item"}
	}
	passwordData, isData := cf.AsData(passwordValue)
	if !isData {
		return "", nil, false, KeychainErrorInvalidResponse{Message: "password was not data"}
	}
	return accountString.GoString(), passwordData.Bytes(), true, nil
}

// DeleteCredentials deletes the item stored for server.
func DeleteCredentials(server string) error {
	if err := loadSecurity(); err != nil {
		return securityUnavailable(err)
	}
	serverRef, err := stringRef("server", server)
	if err != nil {
		return err
	}
	defer serverRef.Release()
	query, err := dictionaryRef(
		[]cf.TypeRef{kSecClass, kSecAttrServer},
		[]cf.TypeRef{kSecClassInternetPassword, serverRef.TypeRef()},
	)
	if err != nil {
		return err
	}
	defer query.Release()
	status := secItemDelete(query.Ref())
	return checkStatus(KeychainOperationDelete, status)
}

func readStatus(status int32) (bool, error) {
	if status == errSecItemNotFound {
		return false, nil
	}
	if err := checkStatus(KeychainOperationRead, status); err != nil {
		return false, err
	}
	return true, nil
}

func createStatus(status int32) (bool, error) {
	if status == errSecDuplicateItem {
		return false, nil
	}
	if err := checkStatus(KeychainOperationCreate, status); err != nil {
		return false, err
	}
	return true, nil
}

func checkStatus(operation KeychainOperation, status int32) error {
	if status == errSecSuccess {
		return nil
	} else if status == errSecUserCanceled {
		return KeychainErrorUserCanceled{Operation: operation}
	} else {
		return KeychainErrorSecurity{Operation: operation, Status: status}
	}
}
