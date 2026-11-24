//go:build linux

package dbus

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"

	godbus "github.com/godbus/dbus/v5"
)

// KeyringLabel is the label every credential item is stored under.
const KeyringLabel = "zed-github-account"

const (
	secretsService         = "org.freedesktop.secrets"
	secretsPath            = godbus.ObjectPath("/org/freedesktop/secrets")
	secretServiceInterface = "org.freedesktop.Secret.Service"
	secretCollectionIface  = "org.freedesktop.Secret.Collection"
	secretItemIface        = "org.freedesktop.Secret.Item"
	secretPromptIface      = "org.freedesktop.Secret.Prompt"
	algorithmPlain         = "plain"
	algorithmEncrypted     = "dh-ietf1024-sha256-aes128-cbc-pkcs7"
	defaultCollectionAlias = "default"
	defaultCollectionLabel = "Default"
	blobContentType        = "application/octet-stream"
	noPrompt               = godbus.ObjectPath("/")
)

// ErrPromptDismissed is returned when the user dismisses a Secret Service prompt.
var ErrPromptDismissed = errors.New("the prompt was dismissed")

var dhPrime = new(big.Int).SetBytes([]byte{
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xC9, 0x0F, 0xDA, 0xA2, 0x21, 0x68,
	0xC2, 0x34, 0xC4, 0xC6, 0x62, 0x8B, 0x80, 0xDC, 0x1C, 0xD1, 0x29, 0x02, 0x4E, 0x08,
	0x8A, 0x67, 0xCC, 0x74, 0x02, 0x0B, 0xBE, 0xA6, 0x3B, 0x13, 0x9B, 0x22, 0x51, 0x4A,
	0x08, 0x79, 0x8E, 0x34, 0x04, 0xDD, 0xEF, 0x95, 0x19, 0xB3, 0xCD, 0x3A, 0x43, 0x1B,
	0x30, 0x2B, 0x0A, 0x6D, 0xF2, 0x5F, 0x14, 0x37, 0x4F, 0xE1, 0x35, 0x6D, 0x6D, 0x51,
	0xC2, 0x45, 0xE4, 0x85, 0xB5, 0x76, 0x62, 0x5E, 0x7E, 0xC6, 0xF4, 0x4C, 0x42, 0xE9,
	0xA6, 0x37, 0xED, 0x6B, 0x0B, 0xFF, 0x5C, 0xB6, 0xF4, 0x06, 0xB7, 0xED, 0xEE, 0x38,
	0x6B, 0xFB, 0x5A, 0x89, 0x9F, 0xA5, 0xAE, 0x9F, 0x24, 0x11, 0x7C, 0x4B, 0x1F, 0xE6,
	0x49, 0x28, 0x66, 0x51, 0xEC, 0xE6, 0x53, 0x81, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF,
})

type secretValue struct {
	Session     godbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string
}

type keyring struct {
	conn       *godbus.Conn
	session    godbus.ObjectPath
	aesKey     []byte
	collection godbus.ObjectPath
}

func openKeyring(conn *godbus.Conn) (*keyring, error) {
	k := &keyring{conn: conn}
	if err := k.openEncrypted(); err != nil {
		var methodErr godbus.Error
		if !errors.As(err, &methodErr) {
			return nil, err
		}
		if err := k.openPlain(); err != nil {
			return nil, err
		}
	}
	collection, err := k.defaultCollection()
	if err != nil {
		k.close()
		return nil, err
	}
	k.collection = collection
	return k, nil
}

func (k *keyring) service() godbus.BusObject { return k.conn.Object(secretsService, secretsPath) }

func (k *keyring) object(path godbus.ObjectPath) godbus.BusObject {
	return k.conn.Object(secretsService, path)
}

func (k *keyring) openPlain() error {
	var output godbus.Variant
	return k.service().Call(secretServiceInterface+".OpenSession", 0, algorithmPlain, godbus.MakeVariant("")).Store(&output, &k.session)
}

func (k *keyring) openEncrypted() error {
	private := make([]byte, 16)
	if _, err := rand.Read(private); err != nil {
		return err
	}
	privateInt := new(big.Int).SetBytes(private)
	public := new(big.Int).Exp(big.NewInt(2), privateInt, dhPrime).Bytes()
	var output godbus.Variant
	if err := k.service().Call(secretServiceInterface+".OpenSession", 0, algorithmEncrypted, godbus.MakeVariant(public)).Store(&output, &k.session); err != nil {
		return err
	}
	serverPublic, ok := output.Value().([]byte)
	if !ok {
		k.close()
		return fmt.Errorf("secret service returned a %s session key", output.Signature())
	}
	key, err := generateAESKey(privateInt, serverPublic)
	if err != nil {
		k.close()
		return err
	}
	k.aesKey = key
	return nil
}

func generateAESKey(private *big.Int, serverPublic []byte) ([]byte, error) {
	common := new(big.Int).Exp(new(big.Int).SetBytes(serverPublic), private, dhPrime).Bytes()
	ikm := make([]byte, 128-len(common), 128)
	ikm = append(ikm, common...)
	return hkdf.Key(sha256.New, ikm, nil, "", 16)
}

func (k *keyring) close() {
	if k.session != "" && k.session != noPrompt {
		_ = k.object(k.session).Call("org.freedesktop.Secret.Session.Close", 0).Err
	}
}

func (k *keyring) defaultCollection() (godbus.ObjectPath, error) {
	var path godbus.ObjectPath
	if err := k.service().Call(secretServiceInterface+".ReadAlias", 0, defaultCollectionAlias).Store(&path); err != nil {
		return "", err
	}
	if path != noPrompt {
		return path, nil
	}
	props := map[string]godbus.Variant{secretCollectionIface + ".Label": godbus.MakeVariant(defaultCollectionLabel)}
	var prompt godbus.ObjectPath
	if err := k.service().Call(secretServiceInterface+".CreateCollection", 0, props, defaultCollectionAlias).Store(&path, &prompt); err != nil {
		return "", err
	}
	if prompt != noPrompt {
		result, err := k.prompt(prompt)
		if err != nil {
			return "", err
		}
		created, ok := result.Value().(godbus.ObjectPath)
		if !ok {
			return "", fmt.Errorf("unexpected prompt result %s", result.Signature())
		}
		path = created
	}
	return path, nil
}

func (k *keyring) prompt(path godbus.ObjectPath) (godbus.Variant, error) {
	signals := make(chan *godbus.Signal, 4)
	k.conn.Signal(signals)
	defer k.conn.RemoveSignal(signals)
	match := []godbus.MatchOption{
		godbus.WithMatchObjectPath(path),
		godbus.WithMatchInterface(secretPromptIface),
		godbus.WithMatchMember("Completed"),
	}
	if err := k.conn.AddMatchSignal(match...); err != nil {
		return godbus.Variant{}, err
	}
	defer func() { _ = k.conn.RemoveMatchSignal(match...) }()
	if err := k.object(path).Call(secretPromptIface+".Prompt", 0, "").Err; err != nil {
		return godbus.Variant{}, err
	}
	for sig := range signals {
		if sig.Path != path || sig.Name != secretPromptIface+".Completed" || len(sig.Body) < 2 {
			continue
		}
		if dismissed, _ := sig.Body[0].(bool); dismissed {
			return godbus.Variant{}, ErrPromptDismissed
		}
		result, _ := sig.Body[1].(godbus.Variant)
		return result, nil
	}
	return godbus.Variant{}, errors.New("connection closed while waiting for the prompt")
}

func (k *keyring) unlock(paths ...godbus.ObjectPath) error {
	var unlocked []godbus.ObjectPath
	var prompt godbus.ObjectPath
	if err := k.service().Call(secretServiceInterface+".Unlock", 0, paths).Store(&unlocked, &prompt); err != nil {
		return err
	}
	if prompt != noPrompt {
		if _, err := k.prompt(prompt); err != nil {
			return err
		}
	}
	return nil
}

func (k *keyring) searchItems(attributes map[string]string) ([]godbus.ObjectPath, error) {
	var items []godbus.ObjectPath
	err := k.object(k.collection).Call(secretCollectionIface+".SearchItems", 0, attributes).Store(&items)
	return items, err
}

func (k *keyring) property(path godbus.ObjectPath, name string, out any) error {
	return k.object(path).Call("org.freedesktop.DBus.Properties.Get", 0, secretItemIface, name).Store(out)
}

func (k *keyring) createItem(label string, attributes map[string]string, secret []byte, replace bool) error {
	value, err := k.encode(secret)
	if err != nil {
		return err
	}
	props := map[string]godbus.Variant{
		secretItemIface + ".Label":      godbus.MakeVariant(label),
		secretItemIface + ".Attributes": godbus.MakeVariant(attributes),
	}
	var item, prompt godbus.ObjectPath
	if err := k.object(k.collection).Call(secretCollectionIface+".CreateItem", 0, props, value, replace).Store(&item, &prompt); err != nil {
		return err
	}
	if prompt != noPrompt {
		if _, err := k.prompt(prompt); err != nil {
			return err
		}
	}
	return nil
}

func (k *keyring) secret(item godbus.ObjectPath) ([]byte, error) {
	var value secretValue
	if err := k.object(item).Call(secretItemIface+".GetSecret", 0, k.session).Store(&value); err != nil {
		return nil, err
	}
	return k.decode(value)
}

func (k *keyring) deleteItem(item godbus.ObjectPath) error {
	var prompt godbus.ObjectPath
	if err := k.object(item).Call(secretItemIface+".Delete", 0).Store(&prompt); err != nil {
		return err
	}
	if prompt != noPrompt {
		_, err := k.prompt(prompt)
		return err
	}
	return nil
}

func (k *keyring) encode(secret []byte) (secretValue, error) {
	if k.aesKey == nil {
		return secretValue{Session: k.session, Parameters: []byte{}, Value: secret, ContentType: blobContentType}, nil
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return secretValue{}, err
	}
	block, err := aes.NewCipher(k.aesKey)
	if err != nil {
		return secretValue{}, err
	}
	pad := aes.BlockSize - len(secret)%aes.BlockSize
	padded := append(append([]byte{}, secret...), bytes.Repeat([]byte{byte(pad)}, pad)...)
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, padded)
	return secretValue{Session: k.session, Parameters: iv, Value: out, ContentType: blobContentType}, nil
}

func (k *keyring) decode(value secretValue) ([]byte, error) {
	if k.aesKey == nil {
		return value.Value, nil
	}
	if len(value.Parameters) != aes.BlockSize || len(value.Value) == 0 || len(value.Value)%aes.BlockSize != 0 {
		return nil, errors.New("Wrong padding error")
	}
	block, err := aes.NewCipher(k.aesKey)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(value.Value))
	cipher.NewCBCDecrypter(block, value.Parameters).CryptBlocks(out, value.Value)
	pad := int(out[len(out)-1])
	if pad == 0 || pad > aes.BlockSize || !bytes.Equal(out[len(out)-pad:], bytes.Repeat([]byte{byte(pad)}, pad)) {
		return nil, errors.New("Wrong padding error")
	}
	return out[:len(out)-pad], nil
}

// WriteCredentials stores password for url and username in the default collection,
// replacing an item with the same attributes.
func WriteCredentials(conn *godbus.Conn, url, username string, password []byte) error {
	k, err := openKeyring(conn)
	if err != nil {
		return err
	}
	defer k.close()
	if err := k.unlock(k.collection); err != nil {
		return err
	}
	return k.createItem(KeyringLabel, map[string]string{"url": url, "username": username}, password, true)
}

// ReadCredentials returns the username and password stored for url; ok is false when none is stored.
func ReadCredentials(conn *godbus.Conn, url string) (username string, password []byte, ok bool, err error) {
	k, err := openKeyring(conn)
	if err != nil {
		return "", nil, false, err
	}
	defer k.close()
	if err := k.unlock(k.collection); err != nil {
		return "", nil, false, err
	}
	items, err := k.searchItems(map[string]string{"url": url})
	if err != nil {
		return "", nil, false, err
	}
	for _, item := range items {
		var label string
		if k.property(item, "Label", &label) != nil || label != KeyringLabel {
			continue
		}
		var attributes map[string]string
		if err := k.property(item, "Attributes", &attributes); err != nil {
			return "", nil, false, err
		}
		name, found := attributes["username"]
		if !found {
			return "", nil, false, errors.New("Cannot find username in stored credentials")
		}
		if err := k.unlock(item); err != nil {
			return "", nil, false, err
		}
		secret, err := k.secret(item)
		if err != nil {
			return "", nil, false, err
		}
		return name, secret, true, nil
	}
	return "", nil, false, nil
}

// DeleteCredentials deletes the first credential item stored for url.
func DeleteCredentials(conn *godbus.Conn, url string) error {
	k, err := openKeyring(conn)
	if err != nil {
		return err
	}
	defer k.close()
	if err := k.unlock(k.collection); err != nil {
		return err
	}
	items, err := k.searchItems(map[string]string{"url": url})
	if err != nil {
		return err
	}
	for _, item := range items {
		var label string
		if k.property(item, "Label", &label) == nil && label == KeyringLabel {
			return k.deleteItem(item)
		}
	}
	return nil
}
