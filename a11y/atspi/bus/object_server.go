package bus

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
)

const (
	propertiesInterface   = "org.freedesktop.DBus.Properties"
	introspectInterface   = "org.freedesktop.DBus.Introspectable"
	peerInterface         = "org.freedesktop.DBus.Peer"
	errorUnknownObject    = "org.freedesktop.DBus.Error.UnknownObject"
	errorFailed           = "org.freedesktop.DBus.Error.Failed"
	errorInvalidArgs      = "org.freedesktop.DBus.Error.InvalidArgs"
	errorUnknownInterface = "org.freedesktop.DBus.Error.UnknownInterface"
	errorUnknownProperty  = "org.freedesktop.DBus.Error.UnknownProperty"
	errorPropertyReadOnly = "org.freedesktop.DBus.Error.PropertyReadOnly"
)

var (
	errInterfaceExists   = errors.New("interface already exists at the given path")
	errMissingParameter  = errors.New("missing parameter")
	errInterfaceNotFound = errors.New("interface not found")
)

func fdoError(name, message string) *dbus.Error {
	return dbus.NewError(name, []any{message})
}

func fdoUnknownObject(message string) *dbus.Error {
	return fdoError(errorUnknownObject, message)
}

func fdoFailed(message string) *dbus.Error {
	return fdoError(errorFailed, message)
}

func fdoInvalidArgs(message string) *dbus.Error {
	return fdoError(errorInvalidArgs, message)
}

type dbusArg struct {
	name      string
	signature string
}

type dbusMethod struct {
	name string
	in   []dbusArg
	out  []dbusArg
	fn   any
}

type dbusProperty struct {
	name      string
	signature string
	get       func() (any, *dbus.Error)
	set       func(dbus.Variant) *dbus.Error
}

type dbusInterface struct {
	name       string
	methods    []dbusMethod
	properties []dbusProperty
}

func (i *dbusInterface) property(name string) (*dbusProperty, bool) {
	for n := range i.properties {
		if i.properties[n].name == name {
			return &i.properties[n], true
		}
	}
	return nil, false
}

type serverObject struct {
	interfaces []*dbusInterface
}

func (o *serverObject) lookup(name string) (*dbusInterface, bool) {
	for _, iface := range o.interfaces {
		if iface.name == name {
			return iface, true
		}
	}
	return nil, false
}

type objectServer struct {
	conn    *dbus.Conn
	mu      sync.Mutex
	objects map[dbus.ObjectPath]*serverObject
}

func newObjectServer(conn *dbus.Conn) *objectServer {
	return &objectServer{conn: conn, objects: make(map[dbus.ObjectPath]*serverObject)}
}

func (s *objectServer) at(path dbus.ObjectPath, iface *dbusInterface) (bool, error) {
	if !path.IsValid() {
		return false, fmt.Errorf("invalid object path %q", path)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	object, ok := s.objects[path]
	created := !ok
	if ok {
		if _, exists := object.lookup(iface.name); exists {
			return false, nil
		}
	} else {
		object = &serverObject{}
		s.objects[path] = object
		if err := s.exportStandard(path); err != nil {
			delete(s.objects, path)
			return false, err
		}
	}
	table := make(map[string]any, len(iface.methods))
	for _, method := range iface.methods {
		table[method.name] = recoverHandler(method.fn)
	}
	if err := s.conn.ExportMethodTable(table, path, iface.name); err != nil {
		if created {
			delete(s.objects, path)
			_ = s.conn.ExportMethodTable(nil, path, propertiesInterface)
			_ = s.conn.ExportMethodTable(nil, path, introspectInterface)
		}
		return false, err
	}
	object.interfaces = append(object.interfaces, iface)
	return true, nil
}

func (s *objectServer) remove(path dbus.ObjectPath, name string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	object, ok := s.objects[path]
	if !ok {
		return false, errInterfaceNotFound
	}
	index := slices.IndexFunc(object.interfaces, func(iface *dbusInterface) bool { return iface.name == name })
	if index < 0 {
		return false, errInterfaceNotFound
	}
	object.interfaces = slices.Delete(object.interfaces, index, index+1)
	if err := s.conn.ExportMethodTable(nil, path, name); err != nil {
		return false, err
	}
	if len(object.interfaces) > 0 {
		return false, nil
	}
	delete(s.objects, path)
	if err := s.conn.ExportMethodTable(nil, path, propertiesInterface); err != nil {
		return true, err
	}
	return true, s.conn.ExportMethodTable(nil, path, introspectInterface)
}

func (s *objectServer) lookup(path dbus.ObjectPath, name string) (*dbusInterface, *dbus.Error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	object, ok := s.objects[path]
	if !ok {
		return nil, fdoError(errorUnknownObject, fmt.Sprintf("Unknown object '%s'", path))
	}
	iface, ok := object.lookup(name)
	if !ok {
		return nil, fdoError(errorUnknownInterface, fmt.Sprintf("Unknown interface '%s'", name))
	}
	return iface, nil
}

func (s *objectServer) exportStandard(path dbus.ObjectPath) error {
	properties := map[string]any{
		"Get": func(interfaceName, propertyName string) (dbus.Variant, *dbus.Error) {
			iface, err := s.lookup(path, interfaceName)
			if err != nil {
				return dbus.Variant{}, err
			}
			property, ok := iface.property(propertyName)
			if !ok {
				return dbus.Variant{}, fdoError(errorUnknownProperty, fmt.Sprintf("Unknown property '%s'", propertyName))
			}
			value, err := property.get()
			if err != nil {
				return dbus.Variant{}, err
			}
			return dbus.MakeVariant(value), nil
		},
		"GetAll": func(interfaceName string) (map[string]dbus.Variant, *dbus.Error) {
			iface, err := s.lookup(path, interfaceName)
			if err != nil {
				return nil, err
			}
			values := make(map[string]dbus.Variant, len(iface.properties))
			for _, property := range iface.properties {
				value, err := property.get()
				if err != nil {
					return nil, err
				}
				values[property.name] = dbus.MakeVariant(value)
			}
			return values, nil
		},
		"Set": func(interfaceName, propertyName string, value dbus.Variant) *dbus.Error {
			iface, err := s.lookup(path, interfaceName)
			if err != nil {
				return err
			}
			property, ok := iface.property(propertyName)
			if !ok {
				return fdoError(errorUnknownProperty, fmt.Sprintf("Unknown property '%s'", propertyName))
			}
			if property.set == nil {
				return fdoError(errorPropertyReadOnly, fmt.Sprintf("Property '%s' is read-only", propertyName))
			}
			if err := property.set(value); err != nil {
				return err
			}
			if current, err := property.get(); err == nil {
				_ = s.conn.Emit(path, propertiesInterface+".PropertiesChanged", interfaceName, map[string]dbus.Variant{propertyName: dbus.MakeVariant(current)}, []string{})
			}
			return nil
		},
	}
	if err := s.conn.ExportMethodTable(recoverTable(properties), path, propertiesInterface); err != nil {
		return err
	}
	introspect := map[string]any{
		"Introspect": func() (string, *dbus.Error) {
			return s.introspect(path), nil
		},
	}
	if err := s.conn.ExportMethodTable(recoverTable(introspect), path, introspectInterface); err != nil {
		_ = s.conn.ExportMethodTable(nil, path, propertiesInterface)
		return err
	}
	return nil
}

var dbusErrorType = reflect.TypeFor[*dbus.Error]()

func recoverTable(table map[string]any) map[string]any {
	for name, fn := range table {
		table[name] = recoverHandler(fn)
	}
	return table
}

func recoverHandler(fn any) any {
	value := reflect.ValueOf(fn)
	typ := value.Type()
	if typ.Kind() != reflect.Func || typ.NumOut() == 0 || typ.Out(typ.NumOut()-1) != dbusErrorType {
		return fn
	}
	return reflect.MakeFunc(typ, func(args []reflect.Value) (results []reflect.Value) {
		defer func() {
			if r := recover(); r != nil {
				results = make([]reflect.Value, typ.NumOut())
				for n := range results {
					results[n] = reflect.Zero(typ.Out(n))
				}
				results[len(results)-1] = reflect.ValueOf(fdoFailed(fmt.Sprint(r)))
			}
		}()
		if typ.IsVariadic() {
			return value.CallSlice(args)
		}
		return value.Call(args)
	}).Interface()
}

func (s *objectServer) introspect(path dbus.ObjectPath) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var xml strings.Builder
	xml.WriteString("<!DOCTYPE node PUBLIC \"-//freedesktop//DTD D-BUS Object Introspection 1.0//EN\"\n \"http://www.freedesktop.org/standards/dbus/1.0/introspect.dtd\">\n<node>\n")
	if object, ok := s.objects[path]; ok {
		writeStandardInterfaces(&xml)
		for _, iface := range object.interfaces {
			writeInterface(&xml, iface)
		}
	}
	prefix := string(path)
	if prefix != "/" {
		prefix += "/"
	}
	children := map[string]struct{}{}
	for other := range s.objects {
		if rest, ok := strings.CutPrefix(string(other), prefix); ok && rest != "" {
			name, _, _ := strings.Cut(rest, "/")
			children[name] = struct{}{}
		}
	}
	names := make([]string, 0, len(children))
	for name := range children {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		fmt.Fprintf(&xml, "  <node name=\"%s\"/>\n", name)
	}
	xml.WriteString("</node>\n")
	return xml.String()
}

func writeStandardInterfaces(xml *strings.Builder) {
	xml.WriteString("  <interface name=\"" + peerInterface + "\">\n    <method name=\"Ping\">\n    </method>\n    <method name=\"GetMachineId\">\n      <arg type=\"s\" direction=\"out\"/>\n    </method>\n  </interface>\n")
	xml.WriteString("  <interface name=\"" + introspectInterface + "\">\n    <method name=\"Introspect\">\n      <arg type=\"s\" direction=\"out\"/>\n    </method>\n  </interface>\n")
	xml.WriteString("  <interface name=\"" + propertiesInterface + "\">\n" +
		"    <method name=\"Get\">\n      <arg name=\"interface_name\" type=\"s\" direction=\"in\"/>\n      <arg name=\"property_name\" type=\"s\" direction=\"in\"/>\n      <arg type=\"v\" direction=\"out\"/>\n    </method>\n" +
		"    <method name=\"Set\">\n      <arg name=\"interface_name\" type=\"s\" direction=\"in\"/>\n      <arg name=\"property_name\" type=\"s\" direction=\"in\"/>\n      <arg name=\"value\" type=\"v\" direction=\"in\"/>\n    </method>\n" +
		"    <method name=\"GetAll\">\n      <arg name=\"interface_name\" type=\"s\" direction=\"in\"/>\n      <arg type=\"a{sv}\" direction=\"out\"/>\n    </method>\n" +
		"    <signal name=\"PropertiesChanged\">\n      <arg name=\"interface_name\" type=\"s\"/>\n      <arg name=\"changed_properties\" type=\"a{sv}\"/>\n      <arg name=\"invalidated_properties\" type=\"as\"/>\n    </signal>\n" +
		"  </interface>\n")
}

func writeInterface(xml *strings.Builder, iface *dbusInterface) {
	fmt.Fprintf(xml, "  <interface name=\"%s\">\n", iface.name)
	for _, method := range iface.methods {
		fmt.Fprintf(xml, "    <method name=\"%s\">\n", method.name)
		for _, arg := range method.in {
			fmt.Fprintf(xml, "      <arg name=\"%s\" type=\"%s\" direction=\"in\"/>\n", arg.name, arg.signature)
		}
		for _, arg := range method.out {
			fmt.Fprintf(xml, "      <arg type=\"%s\" direction=\"out\"/>\n", arg.signature)
		}
		xml.WriteString("    </method>\n")
	}
	for _, property := range iface.properties {
		access := "read"
		if property.set != nil {
			access = "readwrite"
		}
		fmt.Fprintf(xml, "    <property name=\"%s\" type=\"%s\" access=\"%s\"/>\n", property.name, property.signature, access)
	}
	xml.WriteString("  </interface>\n")
}

func out(signatures ...string) []dbusArg {
	args := make([]dbusArg, len(signatures))
	for n, signature := range signatures {
		args[n] = dbusArg{signature: signature}
	}
	return args
}

func in(pairs ...string) []dbusArg {
	args := make([]dbusArg, 0, len(pairs)/2)
	for n := 0; n+1 < len(pairs); n += 2 {
		args = append(args, dbusArg{name: pairs[n], signature: pairs[n+1]})
	}
	return args
}
