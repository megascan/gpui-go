//go:build linux

package dbus

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"testing"

	godbus "github.com/godbus/dbus/v5"
)

type fakeItem struct {
	path       godbus.ObjectPath
	label      string
	attributes map[string]string
	secret     []byte
}

type fakeSecretService struct {
	t          *testing.T
	conn       *godbus.Conn
	plainOnly  bool
	mu         sync.Mutex
	sessions   map[godbus.ObjectPath]*keyring
	items      []*fakeItem
	next       int
	collection godbus.ObjectPath
	unlocks    int
}

func (s *fakeSecretService) OpenSession(algorithm string, input godbus.Variant) (godbus.Variant, godbus.ObjectPath, *godbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	path := godbus.ObjectPath(fmt.Sprintf("/org/freedesktop/secrets/session/%d", s.next))
	switch algorithm {
	case algorithmPlain:
		s.sessions[path] = &keyring{session: path}
		return godbus.MakeVariant(""), path, nil
	case algorithmEncrypted:
		if s.plainOnly {
			return godbus.Variant{}, "/", godbus.NewError("org.freedesktop.DBus.Error.NotSupported", []any{"no"})
		}
		private := make([]byte, 16)
		_, _ = rand.Read(private)
		privateInt := new(big.Int).SetBytes(private)
		public := new(big.Int).Exp(big.NewInt(2), privateInt, dhPrime).Bytes()
		key, err := generateAESKey(privateInt, input.Value().([]byte))
		if err != nil {
			return godbus.Variant{}, "/", godbus.MakeFailedError(err)
		}
		s.sessions[path] = &keyring{session: path, aesKey: key}
		return godbus.MakeVariant(public), path, nil
	}
	return godbus.Variant{}, "/", godbus.NewError("org.freedesktop.DBus.Error.NotSupported", []any{algorithm})
}

func (s *fakeSecretService) ReadAlias(name string) (godbus.ObjectPath, *godbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == defaultCollectionAlias && s.collection != "" {
		return s.collection, nil
	}
	return "/", nil
}

func (s *fakeSecretService) CreateCollection(props map[string]godbus.Variant, alias string) (godbus.ObjectPath, godbus.ObjectPath, *godbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if alias != defaultCollectionAlias || props[secretCollectionIface+".Label"].Value() != defaultCollectionLabel {
		return "/", "/", godbus.MakeFailedError(fmt.Errorf("bad collection %v %q", props, alias))
	}
	s.collection = "/org/freedesktop/secrets/collection/default"
	if err := s.conn.Export(fakeCollection{s}, s.collection, secretCollectionIface); err != nil {
		s.t.Error(err)
	}
	return s.collection, "/", nil
}

func (s *fakeSecretService) Unlock(objects []godbus.ObjectPath) ([]godbus.ObjectPath, godbus.ObjectPath, *godbus.Error) {
	s.mu.Lock()
	s.unlocks++
	s.mu.Unlock()
	return objects, "/", nil
}

func (s *fakeSecretService) session(path godbus.ObjectPath) (*keyring, *godbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k, ok := s.sessions[path]
	if !ok {
		return nil, godbus.NewError("org.freedesktop.Secret.Error.NoSession", nil)
	}
	return k, nil
}

type fakeCollection struct{ s *fakeSecretService }

func (c fakeCollection) SearchItems(attributes map[string]string) ([]godbus.ObjectPath, *godbus.Error) {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()
	var out []godbus.ObjectPath
	for _, item := range c.s.items {
		match := true
		for k, v := range attributes {
			if item.attributes[k] != v {
				match = false
			}
		}
		if match {
			out = append(out, item.path)
		}
	}
	return out, nil
}

func (c fakeCollection) CreateItem(props map[string]godbus.Variant, secret secretValue, replace bool) (godbus.ObjectPath, godbus.ObjectPath, *godbus.Error) {
	k, derr := c.s.session(secret.Session)
	if derr != nil {
		return "/", "/", derr
	}
	plain, err := k.decode(secret)
	if err != nil {
		return "/", "/", godbus.MakeFailedError(err)
	}
	if secret.ContentType != blobContentType {
		return "/", "/", godbus.MakeFailedError(fmt.Errorf("content type %q", secret.ContentType))
	}
	label, _ := props[secretItemIface+".Label"].Value().(string)
	attributes, _ := props[secretItemIface+".Attributes"].Value().(map[string]string)
	c.s.mu.Lock()
	defer c.s.mu.Unlock()
	if replace {
		for _, item := range c.s.items {
			if fmt.Sprint(item.attributes) == fmt.Sprint(attributes) {
				item.label, item.secret = label, plain
				return item.path, "/", nil
			}
		}
	}
	c.s.next++
	item := &fakeItem{path: godbus.ObjectPath(fmt.Sprintf("%s/%d", c.s.collection, c.s.next)), label: label, attributes: attributes, secret: plain}
	c.s.items = append(c.s.items, item)
	if err := c.s.conn.Export(fakeItemObject{c.s, item}, item.path, secretItemIface); err != nil {
		c.s.t.Error(err)
	}
	props2 := fakeProperties{get: func(iface, name string) (godbus.Variant, bool) {
		c.s.mu.Lock()
		defer c.s.mu.Unlock()
		switch name {
		case "Label":
			return godbus.MakeVariant(item.label), iface == secretItemIface
		case "Attributes":
			return godbus.MakeVariant(item.attributes), iface == secretItemIface
		}
		return godbus.Variant{}, false
	}}
	if err := c.s.conn.Export(props2, item.path, "org.freedesktop.DBus.Properties"); err != nil {
		c.s.t.Error(err)
	}
	return item.path, "/", nil
}

type fakeItemObject struct {
	s    *fakeSecretService
	item *fakeItem
}

func (o fakeItemObject) GetSecret(session godbus.ObjectPath) (secretValue, *godbus.Error) {
	k, derr := o.s.session(session)
	if derr != nil {
		return secretValue{}, derr
	}
	o.s.mu.Lock()
	secret := o.item.secret
	o.s.mu.Unlock()
	v, err := k.encode(secret)
	if err != nil {
		return secretValue{}, godbus.MakeFailedError(err)
	}
	return v, nil
}

func (o fakeItemObject) Delete() (godbus.ObjectPath, *godbus.Error) {
	o.s.mu.Lock()
	defer o.s.mu.Unlock()
	for i, item := range o.s.items {
		if item == o.item {
			o.s.items = append(o.s.items[:i], o.s.items[i+1:]...)
		}
	}
	return "/", nil
}

func TestSecretService_CredentialsRoundTrip(t *testing.T) {
	for _, plainOnly := range []bool{false, true} {
		t.Run(fmt.Sprintf("plainOnly=%v", plainOnly), func(t *testing.T) {
			address := startTestBus(t)
			server := connectTestBus(t, address)
			fake := &fakeSecretService{t: t, conn: server, plainOnly: plainOnly, sessions: map[godbus.ObjectPath]*keyring{}}
			if err := server.Export(fake, secretsPath, secretServiceInterface); err != nil {
				t.Fatal(err)
			}
			if _, err := server.RequestName(secretsService, godbus.NameFlagDoNotQueue); err != nil {
				t.Fatal(err)
			}
			conn := connectTestBus(t, address)
			const url = "https://example.invalid"
			if _, _, ok, err := ReadCredentials(conn, url); err != nil || ok {
				t.Fatalf("ReadCredentials before write = %v, %v", ok, err)
			}
			if err := WriteCredentials(conn, url, "user", []byte("first")); err != nil {
				t.Fatal(err)
			}
			password := bytes.Repeat([]byte{0x10}, 32)
			if err := WriteCredentials(conn, url, "user", password); err != nil {
				t.Fatal(err)
			}
			username, got, ok, err := ReadCredentials(conn, url)
			if err != nil || !ok || username != "user" || !bytes.Equal(got, password) {
				t.Fatalf("ReadCredentials = %q, %q, %v, %v", username, got, ok, err)
			}
			fake.mu.Lock()
			if len(fake.items) != 1 || fake.items[0].label != KeyringLabel {
				t.Fatalf("items = %+v", fake.items)
			}
			fake.mu.Unlock()
			if err := DeleteCredentials(conn, url); err != nil {
				t.Fatal(err)
			}
			if _, _, ok, err := ReadCredentials(conn, url); err != nil || ok {
				t.Fatalf("ReadCredentials after delete = %v, %v", ok, err)
			}
			fake.mu.Lock()
			defer fake.mu.Unlock()
			if fake.unlocks == 0 {
				t.Fatal("collection was never unlocked")
			}
		})
	}
}
