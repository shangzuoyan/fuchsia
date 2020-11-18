// Copyright 2018 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package fidlgen_cpp

import (
	"fmt"
	"sort"
	"strings"

	fidl "go.fuchsia.dev/fuchsia/tools/fidl/lib/fidlgen"
)

// This value needs to be kept in sync with the one defined in
// zircon/system/ulib/fidl/include/lib/fidl/llcpp/sync_call.h
const llcppMaxStackAllocSize = 512

// These are used in header/impl templates to select the correct type-specific template
type bitsKind struct{}
type constKind struct{}
type enumKind struct{}
type protocolKind struct{}
type serviceKind struct{}
type structKind struct{}
type tableKind struct{}
type unionKind struct{}

var Kinds = struct {
	Const    constKind
	Bits     bitsKind
	Enum     enumKind
	Protocol protocolKind
	Service  serviceKind
	Struct   structKind
	Table    tableKind
	Union    unionKind
}{}

type Decl interface {
	isResource() bool
	compileProperties(c compiler)
}

type FamilyKind string

const (
	// TrivialCopy identifies values for whom a copy is trivial (like integers)
	TrivialCopy FamilyKind = "TrivialCopy"
	// Reference identifies values with a non trivial copy for which we use a reference on the
	// caller argument.
	Reference FamilyKind = "Reference"
	// String identifies string values for which we can use a const reference and for which we can
	// optimize the field construction.
	String FamilyKind = "String"
	// Vector identifies vector values for which we can use a reference and for which we can
	// optimize the field construction.
	Vector FamilyKind = "Vector"
)

type TypeKind string

const (
	ArrayKind     TypeKind = "Array"
	VectorKind    TypeKind = "Vector"
	StringKind    TypeKind = "String"
	HandleKind    TypeKind = "Handle"
	RequestKind   TypeKind = "Request"
	PrimitiveKind TypeKind = "Primitive"
	BitsKind      TypeKind = "Bits"
	EnumKind      TypeKind = "Enum"
	ConstKind     TypeKind = "Const"
	StructKind    TypeKind = "Struct"
	TableKind     TypeKind = "Table"
	UnionKind     TypeKind = "Union"
	ProtocolKind  TypeKind = "Protocol"
)

type Type struct {
	Decl string

	// Decl but with full type name
	FullDecl string

	LLDecl    string
	LLClass   string
	LLPointer bool

	// Defines what operation we should use to pass a value without a move (LLCPP). It also
	// defines the way we should initialize a field.
	LLFamily FamilyKind

	// NeedsDtor indicates whether this type needs to be destructed explicitely
	// or not.
	NeedsDtor bool

	Kind TypeKind

	IsResource          bool
	ExternalDeclaration bool

	DeclarationName fidl.EncodedCompoundIdentifier

	// Set iff IsArray || IsVector
	ElementType *Type
	// Valid iff IsArray
	ElementCount int
}

func (t Type) IsPrimitiveType() bool {
	return t.Kind == PrimitiveKind || t.Kind == BitsKind || t.Kind == EnumKind
}

func (t *Type) compileProperties(c compiler) {
	if t.DeclarationName != "" {
		declaration, ok := c.declarations[t.DeclarationName]
		if !ok {
			// The declaration comes from another library.
			t.ExternalDeclaration = true
		} else {
			t.IsResource = declaration.isResource()
		}
	} else {
		switch t.Kind {
		case ArrayKind:
			fallthrough
		case VectorKind:
			t.ElementType.compileProperties(c)
			t.IsResource = t.ElementType.IsResource
			t.ExternalDeclaration = t.ElementType.ExternalDeclaration
			t.LLClass = t.ElementType.LLClass
			t.LLPointer = t.ElementType.LLPointer
		case HandleKind:
			fallthrough
		case RequestKind:
			fallthrough
		case ProtocolKind:
			t.IsResource = true
		}
	}
}

type Const struct {
	fidl.Attributes
	Extern    bool
	Decorator string
	Type      Type
	Name      string
	Value     string

	// Kind should be default initialized.
	Kind constKind
}

func (c Const) isResource() bool {
	return false
}

func (co *Const) compileProperties(c compiler) {
}

type Bits struct {
	fidl.Bits
	Namespace string
	Type      string
	Name      string
	Mask      string
	MaskName  string
	Members   []BitsMember

	// Kind should be default initialized.
	Kind bitsKind
}

func (b Bits) isResource() bool {
	return false
}

func (b *Bits) compileProperties(c compiler) {
}

type BitsMember struct {
	fidl.Attributes
	Name  string
	Value string
}

type Enum struct {
	fidl.Enum
	Namespace string
	Type      string
	Name      string
	Members   []EnumMember

	// Kind should be default initialized.
	Kind enumKind
}

func (e Enum) isResource() bool {
	return false
}

func (e *Enum) compileProperties(c compiler) {
}

type EnumMember struct {
	fidl.EnumMember
	Name  string
	Value string
}

type Union struct {
	fidl.Union
	Namespace    string
	Name         string
	TableType    string
	Members      []UnionMember
	InlineSize   int
	MaxHandles   int
	MaxOutOfLine int
	Result       *Result
	HasPointer   bool
	IsResource   bool

	// Kind should be default initialized.
	Kind unionKind
}

func (u Union) isResource() bool {
	return u.IsResource
}

func (u *Union) compileProperties(c compiler) {
	for i := 0; i < len(u.Members); i++ {
		u.Members[i].Type.compileProperties(c)
	}
}

type UnionMember struct {
	fidl.Attributes
	Ordinal     uint64
	Type        Type
	Name        string
	StorageName string
	TagName     string
	Offset      int
}

func (um UnionMember) UpperCamelCaseName() string {
	return fidl.ToUpperCamelCase(um.Name)
}

type TableFrameItem struct {
	LLDecl string // "void" if reserved
	Name   string
}

type Table struct {
	fidl.Table
	Namespace      string
	Name           string
	TableType      string
	Members        []TableMember
	InlineSize     int
	BiggestOrdinal int
	MaxHandles     int
	MaxOutOfLine   int
	MaxSentSize    int
	HasPointer     bool
	IsResource     bool
	// Types of the members in ordinal order, "void" for reserved.
	FrameItems []TableFrameItem

	// Kind should be default initialized.
	Kind tableKind
}

func (t Table) isResource() bool {
	return t.IsResource
}

func (t *Table) compileProperties(c compiler) {
	for i := 0; i < len(t.Members); i++ {
		t.Members[i].Type.compileProperties(c)
	}
}

type TableMember struct {
	fidl.Attributes
	Type               Type
	Name               string
	DefaultValue       string
	Ordinal            int
	FieldPresenceIsSet string
	FieldPresenceSet   string
	FieldPresenceClear string
	FieldDataName      string
	MethodHasName      string
	MethodClearName    string
	ValueUnionName     string
}

type Struct struct {
	fidl.Struct
	Namespace     string
	Name          string
	TableType     string
	Members       []StructMember
	InlineSize    int
	MaxHandles    int
	MaxOutOfLine  int
	MaxSentSize   int
	HasPadding    bool
	IsResultValue bool
	HasPointer    bool
	IsResource    bool
	Result        *Result
	// Full decls needed to check if a type is memcpy compatible.
	// Only set if it may be possible for a type to be memcpy compatible,
	// e.g. has no padding.
	// See the struct template for usage.
	FullDeclMemcpyCompatibleDeps []string

	// Kind should be default initialized.
	Kind structKind
}

func (s Struct) isResource() bool {
	return s.IsResource
}

func (s *Struct) compileProperties(c compiler) {
	for i := 0; i < len(s.Members); i++ {
		s.Members[i].Type.compileProperties(c)
	}
}

type StructMember struct {
	fidl.Attributes
	Type         Type
	Name         string
	DefaultValue string
	Offset       int
}

// protocolInner contains information about a Protocol that should be
// filled out by the compiler.
type protocolInner struct {
	fidl.Attributes
	Namespace           string
	Name                string
	ClassName           string
	ServiceName         string
	ProxyName           string
	StubName            string
	EventSenderName     string
	SyncName            string
	SyncProxyName       string
	RequestEncoderName  string
	RequestDecoderName  string
	ResponseEncoderName string
	ResponseDecoderName string
	Methods             []Method
}

// Protocol should be created using protocolInner.build().
type Protocol struct {
	protocolInner

	// OneWayMethods contains the list of one-way (i.e. fire-and-forget) methods
	// in the protocol.
	OneWayMethods []Method

	// TwoWayMethods contains the list of two-way (i.e. has both request and
	// response) methods in the protocol.
	TwoWayMethods []Method

	// ClientMethods contains the list of client-initiated methods (i.e. any
	// interaction that is not an event). It is the union of one-way and two-way
	// methods.
	ClientMethods []Method

	// Events contains the list of events (i.e. initiated by servers)
	// in the protocol.
	Events []Method

	// Kind should always be default initialized.
	Kind protocolKind
}

func (inner protocolInner) build() *Protocol {
	type kinds []methodKind

	filterBy := func(kinds kinds) []Method {
		var out []Method
		for _, m := range inner.Methods {
			k := m.methodKind()
			for _, want := range kinds {
				if want == k {
					out = append(out, m)
				}
			}
		}
		return out
	}

	return &Protocol{
		protocolInner: inner,
		OneWayMethods: filterBy(kinds{oneWayMethod}),
		TwoWayMethods: filterBy(kinds{twoWayMethod}),
		ClientMethods: filterBy(kinds{oneWayMethod, twoWayMethod}),
		Events:        filterBy(kinds{eventMethod}),
	}
}

func (p *Protocol) isResource() bool {
	return false
}

func (p *Protocol) compileProperties(c compiler) {
	for i := 0; i < len(p.Methods); i++ {
		p.Methods[i].compileProperties(c)
	}
}

type Service struct {
	fidl.Attributes
	Namespace   string
	Name        string
	ServiceName string
	Members     []ServiceMember

	// Kind should be default initialized.
	Kind serviceKind
}

func (s Service) isResource() bool {
	return false
}

func (s *Service) compileProperties(c compiler) {
}

type ServiceMember struct {
	fidl.Attributes
	ProtocolType string
	Name         string
	MethodName   string
}

// TODO: There are common fields between Request and Response; consider factoring them out.
type Method struct {
	fidl.Attributes
	Name                 string
	NameInLowerSnakeCase string
	Ordinal              uint64
	// The name of a constant that defines the ordinal value.
	OrdinalName             string
	HasRequest              bool
	Request                 []Parameter
	RequestSize             int
	RequestTypeName         string
	RequestMaxHandles       int
	RequestMaxOutOfLine     int
	RequestSentMaxSize      int
	RequestPadding          bool
	RequestFlexible         bool
	RequestHasPointer       bool
	RequestIsResource       bool
	HasResponse             bool
	Response                []Parameter
	ResponseSize            int
	ResponseTypeName        string
	ResponseMaxHandles      int
	ResponseMaxOutOfLine    int
	ResponseSentMaxSize     int
	ResponseReceivedMaxSize int
	ResponsePadding         bool
	ResponseFlexible        bool
	ResponseHasPointer      bool
	ResponseIsResource      bool
	CallbackType            string
	ResponseHandlerType     string
	ResponderType           string
	Transitional            bool
	Result                  *Result
	LLProps                 LLProps
}

func (m *Method) compileProperties(c compiler) {
	for i := 0; i < len(m.Request); i++ {
		m.Request[i].Type.compileProperties(c)
	}
	for i := 0; i < len(m.Response); i++ {
		m.Response[i].Type.compileProperties(c)
	}
}

type methodKind int

const (
	oneWayMethod = methodKind(iota)
	twoWayMethod
	eventMethod
)

func (m *Method) methodKind() methodKind {
	if m.HasRequest {
		if m.HasResponse {
			return twoWayMethod
		}
		return oneWayMethod
	}
	if !m.HasResponse {
		panic("A method should have at least either a request or a response")
	}
	return eventMethod
}

// LLContextProps contain context-dependent properties of a method specific to llcpp.
// Context is client (write request and read response) or server (read request and write response).
type LLContextProps struct {
	// Should the request be allocated on the stack, in the managed flavor.
	StackAllocRequest bool
	// Should the response be allocated on the stack, in the managed flavor.
	StackAllocResponse bool
	// Total number of bytes of stack used for storing the request.
	StackUseRequest int
	// Total number of bytes of stack used for storing the response.
	StackUseResponse int
}

// LLProps contain properties of a method specific to llcpp
type LLProps struct {
	ProtocolName      string
	LinearizeRequest  bool
	LinearizeResponse bool
	ClientContext     LLContextProps
	ServerContext     LLContextProps
}

type Parameter struct {
	Type   Type
	Name   string
	Offset int
}

type Root struct {
	PrimaryHeader   string
	IncludeStem     string
	Headers         []string
	FuzzerHeaders   []string
	HandleTypes     []string
	Library         fidl.LibraryIdentifier
	LibraryReversed fidl.LibraryIdentifier
	Decls           []Decl
}

// Holds information about error results on methods
type Result struct {
	ValueMembers    []StructMember
	ResultDecl      string
	ErrorDecl       string
	ValueDecl       string
	ValueStructDecl string
	ValueTupleDecl  string
}

func (r Result) ValueArity() int {
	return len(r.ValueMembers)
}

func (m *Method) CallbackWrapper() string {
	return "fit::function"
}

var reservedWords = map[string]struct{}{
	"alignas":          {},
	"alignof":          {},
	"and":              {},
	"and_eq":           {},
	"asm":              {},
	"assert":           {},
	"atomic_cancel":    {},
	"atomic_commit":    {},
	"atomic_noexcept":  {},
	"auto":             {},
	"bitand":           {},
	"bitor":            {},
	"bool":             {},
	"break":            {},
	"case":             {},
	"catch":            {},
	"char":             {},
	"char16_t":         {},
	"char32_t":         {},
	"class":            {},
	"compl":            {},
	"concept":          {},
	"const":            {},
	"constexpr":        {},
	"const_cast":       {},
	"continue":         {},
	"co_await":         {},
	"co_return":        {},
	"co_yield":         {},
	"decltype":         {},
	"default":          {},
	"delete":           {},
	"do":               {},
	"double":           {},
	"dynamic_cast":     {},
	"else":             {},
	"enum":             {},
	"explicit":         {},
	"export":           {},
	"extern":           {},
	"false":            {},
	"float":            {},
	"for":              {},
	"friend":           {},
	"goto":             {},
	"if":               {},
	"import":           {},
	"inline":           {},
	"int":              {},
	"long":             {},
	"module":           {},
	"mutable":          {},
	"namespace":        {},
	"new":              {},
	"noexcept":         {},
	"not":              {},
	"not_eq":           {},
	"NULL":             {},
	"nullptr":          {},
	"offsetof":         {},
	"operator":         {},
	"or":               {},
	"or_eq":            {},
	"private":          {},
	"protected":        {},
	"public":           {},
	"register":         {},
	"reinterpret_cast": {},
	"requires":         {},
	"return":           {},
	"short":            {},
	"signed":           {},
	"sizeof":           {},
	"static":           {},
	"static_assert":    {},
	"static_cast":      {},
	"struct":           {},
	"switch":           {},
	"synchronized":     {},
	"template":         {},
	"this":             {},
	"thread_local":     {},
	"throw":            {},
	"true":             {},
	"try":              {},
	"typedef":          {},
	"typeid":           {},
	"typename":         {},
	"union":            {},
	"unsigned":         {},
	"using":            {},
	"virtual":          {},
	"void":             {},
	"volatile":         {},
	"wchar_t":          {},
	"while":            {},
	"xor":              {},
	"xor_eq":           {},
	"xunion":           {},

	// names used in specific contexts e.g. union accessors
	"FidlType":        {},
	"New":             {},
	"Tag":             {},
	"Which":           {},
	"has_invalid_tag": {},
	"which":           {},
	"Unknown":         {},
	"unknown":         {},
	"UnknownBytes":    {},
	"UnknownData":     {},
	"IsEmpty":         {},
	"HandleEvents":    {},
	// TODO(ianloic) add: "Clone"
	// There are Clone methods on a couple of protocols that are used
	// across layers so this will be a breaking change.
	// fxbug.dev/7785

	// All names from errno definitions.
	"EPERM":           {},
	"ENOENT":          {},
	"ESRCH":           {},
	"EINTR":           {},
	"EIO":             {},
	"ENXIO":           {},
	"E2BIG":           {},
	"ENOEXEC":         {},
	"EBADF":           {},
	"ECHILD":          {},
	"EAGAIN":          {},
	"ENOMEM":          {},
	"EACCES":          {},
	"EFAULT":          {},
	"ENOTBLK":         {},
	"EBUSY":           {},
	"EEXIST":          {},
	"EXDEV":           {},
	"ENODEV":          {},
	"ENOTDIR":         {},
	"EISDIR":          {},
	"EINVAL":          {},
	"ENFILE":          {},
	"EMFILE":          {},
	"ENOTTY":          {},
	"ETXTBSY":         {},
	"EFBIG":           {},
	"ENOSPC":          {},
	"ESPIPE":          {},
	"EROFS":           {},
	"EMLINK":          {},
	"EPIPE":           {},
	"EDOM":            {},
	"ERANGE":          {},
	"EDEADLK":         {},
	"ENAMETOOLONG":    {},
	"ENOLCK":          {},
	"ENOSYS":          {},
	"ENOTEMPTY":       {},
	"ELOOP":           {},
	"EWOULDBLOCK":     {},
	"ENOMSG":          {},
	"EIDRM":           {},
	"ECHRNG":          {},
	"EL2NSYNC":        {},
	"EL3HLT":          {},
	"EL3RST":          {},
	"ELNRNG":          {},
	"EUNATCH":         {},
	"ENOCSI":          {},
	"EL2HLT":          {},
	"EBADE":           {},
	"EBADR":           {},
	"EXFULL":          {},
	"ENOANO":          {},
	"EBADRQC":         {},
	"EBADSLT":         {},
	"EDEADLOCK":       {},
	"EBFONT":          {},
	"ENOSTR":          {},
	"ENODATA":         {},
	"ETIME":           {},
	"ENOSR":           {},
	"ENONET":          {},
	"ENOPKG":          {},
	"EREMOTE":         {},
	"ENOLINK":         {},
	"EADV":            {},
	"ESRMNT":          {},
	"ECOMM":           {},
	"EPROTO":          {},
	"EMULTIHOP":       {},
	"EDOTDOT":         {},
	"EBADMSG":         {},
	"EOVERFLOW":       {},
	"ENOTUNIQ":        {},
	"EBADFD":          {},
	"EREMCHG":         {},
	"ELIBACC":         {},
	"ELIBBAD":         {},
	"ELIBSCN":         {},
	"ELIBMAX":         {},
	"ELIBEXEC":        {},
	"EILSEQ":          {},
	"ERESTART":        {},
	"ESTRPIPE":        {},
	"EUSERS":          {},
	"ENOTSOCK":        {},
	"EDESTADDRREQ":    {},
	"EMSGSIZE":        {},
	"EPROTOTYPE":      {},
	"ENOPROTOOPT":     {},
	"EPROTONOSUPPORT": {},
	"ESOCKTNOSUPPORT": {},
	"EOPNOTSUPP":      {},
	"ENOTSUP":         {},
	"EPFNOSUPPORT":    {},
	"EAFNOSUPPORT":    {},
	"EADDRINUSE":      {},
	"EADDRNOTAVAIL":   {},
	"ENETDOWN":        {},
	"ENETUNREACH":     {},
	"ENETRESET":       {},
	"ECONNABORTED":    {},
	"ECONNRESET":      {},
	"ENOBUFS":         {},
	"EISCONN":         {},
	"ENOTCONN":        {},
	"ESHUTDOWN":       {},
	"ETOOMANYREFS":    {},
	"ETIMEDOUT":       {},
	"ECONNREFUSED":    {},
	"EHOSTDOWN":       {},
	"EHOSTUNREACH":    {},
	"EALREADY":        {},
	"EINPROGRESS":     {},
	"ESTALE":          {},
	"EUCLEAN":         {},
	"ENOTNAM":         {},
	"ENAVAIL":         {},
	"EISNAM":          {},
	"EREMOTEIO":       {},
	"EDQUOT":          {},
	"ENOMEDIUM":       {},
	"EMEDIUMTYPE":     {},
	"ECANCELED":       {},
	"ENOKEY":          {},
	"EKEYEXPIRED":     {},
	"EKEYREVOKED":     {},
	"EKEYREJECTED":    {},
	"EOWNERDEAD":      {},
	"ENOTRECOVERABLE": {},
	"ERFKILL":         {},
	"EHWPOISON":       {},
}

var primitiveTypes = map[fidl.PrimitiveSubtype]string{
	fidl.Bool:    "bool",
	fidl.Int8:    "int8_t",
	fidl.Int16:   "int16_t",
	fidl.Int32:   "int32_t",
	fidl.Int64:   "int64_t",
	fidl.Uint8:   "uint8_t",
	fidl.Uint16:  "uint16_t",
	fidl.Uint32:  "uint32_t",
	fidl.Uint64:  "uint64_t",
	fidl.Float32: "float",
	fidl.Float64: "double",
}

func isReservedWord(str string) bool {
	_, ok := reservedWords[str]
	return ok
}

func changeIfReserved(i fidl.Identifier, ext string) string {
	str := string(i) + ext
	if isReservedWord(str) {
		return str + "_"
	}
	return str
}

type identifierTransform bool

const (
	keepPartIfReserved   identifierTransform = false
	changePartIfReserved identifierTransform = true
)

func formatLibrary(library fidl.LibraryIdentifier, sep string, identifierTransform identifierTransform) string {
	parts := []string{}
	for _, part := range library {
		if identifierTransform == changePartIfReserved {
			parts = append(parts, changeIfReserved(part, ""))
		} else {
			parts = append(parts, string(part))
		}
	}
	name := strings.Join(parts, sep)
	return changeIfReserved(fidl.Identifier(name), "")
}

func formatNamespace(library fidl.LibraryIdentifier, appendNamespace string) string {
	ns := "::" + formatLibrary(library, "::", changePartIfReserved)
	if len(appendNamespace) > 0 {
		ns = ns + "::" + appendNamespace
	}
	return ns
}

func formatLLNamespace(library fidl.LibraryIdentifier, appendNamespace string) string {
	// Avoid user-defined llcpp library colliding with the llcpp namespace, by appending underscore.
	if len(library) > 0 && library[0] == "llcpp" {
		libraryRenamed := make([]fidl.Identifier, len(library))
		copy(libraryRenamed, library)
		libraryRenamed[0] = "llcpp_"
		library = libraryRenamed
	}
	return "::llcpp" + formatNamespace(library, appendNamespace)
}

func formatLibraryPrefix(library fidl.LibraryIdentifier) string {
	return formatLibrary(library, "_", keepPartIfReserved)
}

func formatLibraryPath(library fidl.LibraryIdentifier) string {
	return formatLibrary(library, "/", keepPartIfReserved)
}

type compiler struct {
	namespace          string
	symbolPrefix       string
	decls              fidl.DeclInfoMap
	declarations       map[fidl.EncodedCompoundIdentifier]Decl
	library            fidl.LibraryIdentifier
	handleTypes        map[fidl.HandleSubtype]struct{}
	namespaceFormatter func(fidl.LibraryIdentifier, string) string
	resultForStruct    map[fidl.EncodedCompoundIdentifier]*Result
	resultForUnion     map[fidl.EncodedCompoundIdentifier]*Result
}

func (c *compiler) isInExternalLibrary(ci fidl.CompoundIdentifier) bool {
	if len(ci.Library) != len(c.library) {
		return true
	}
	for i, part := range c.library {
		if ci.Library[i] != part {
			return true
		}
	}
	return false
}

func (c *compiler) compileCompoundIdentifier(eci fidl.EncodedCompoundIdentifier, ext, appendNamespace string, fullName bool) string {
	val := fidl.ParseCompoundIdentifier(eci)
	strs := []string{}
	if fullName || c.isInExternalLibrary(val) {
		strs = append(strs, c.namespaceFormatter(val.Library, appendNamespace))
	}
	strs = append(strs, changeIfReserved(val.Name, ext))
	if len(val.Member) != 0 {
		strs = append(strs, changeIfReserved(val.Member, ext))
	}
	return strings.Join(strs, "::")
}

func (c *compiler) compileTableType(eci fidl.EncodedCompoundIdentifier) string {
	val := fidl.ParseCompoundIdentifier(eci)
	if c.isInExternalLibrary(val) {
		panic(fmt.Sprintf("can't create table type for external identifier: %v", val))
	}

	return fmt.Sprintf("%s_%sTable", c.symbolPrefix, val.Name)
}

func (c *compiler) compileLiteral(val fidl.Literal, typ fidl.Type) string {
	switch val.Kind {
	case fidl.StringLiteral:
		return fmt.Sprintf("%q", val.Value)
	case fidl.NumericLiteral:
		if val.Value == "-9223372036854775808" || val.Value == "0x8000000000000000" {
			// C++ only supports nonnegative literals and a value this large in absolute
			// value cannot be represented as a nonnegative number in 64-bits.
			return "(-9223372036854775807ll-1)"
		}
		// TODO(fxbug.dev/7810): Once we expose resolved constants for defaults, e.g.
		// in structs, we will not need ignore hex and binary values.
		if strings.HasPrefix(val.Value, "0x") || strings.HasPrefix(val.Value, "0b") {
			return val.Value
		}

		// float32 literals must be marked as such.
		if strings.ContainsRune(val.Value, '.') {
			if typ.Kind == fidl.PrimitiveType && typ.PrimitiveSubtype == fidl.Float32 {
				return fmt.Sprintf("%sf", val.Value)
			} else {
				return val.Value
			}
		}

		if !strings.HasPrefix(val.Value, "-") {
			return fmt.Sprintf("%su", val.Value)
		}
		return val.Value
	case fidl.TrueLiteral:
		return "true"
	case fidl.FalseLiteral:
		return "false"
	case fidl.DefaultLiteral:
		return "default"
	default:
		panic(fmt.Sprintf("unknown literal kind: %v", val.Kind))
	}
}

func (c *compiler) compileConstant(val fidl.Constant, t *Type, typ fidl.Type, appendNamespace string) string {
	switch val.Kind {
	case fidl.IdentifierConstant:
		return c.compileCompoundIdentifier(val.Identifier, "", appendNamespace, false)
	case fidl.LiteralConstant:
		return c.compileLiteral(val.Literal, typ)
	default:
		panic(fmt.Sprintf("unknown constant kind: %v", val.Kind))
	}
}

func (c *compiler) compilePrimitiveSubtype(val fidl.PrimitiveSubtype) string {
	if t, ok := primitiveTypes[val]; ok {
		return t
	}
	panic(fmt.Sprintf("unknown primitive type: %v", val))
}

func (c *compiler) compileType(val fidl.Type) Type {
	r := Type{}
	switch val.Kind {
	case fidl.ArrayType:
		t := c.compileType(*val.ElementType)
		r.Decl = fmt.Sprintf("::std::array<%s, %v>", t.Decl, *val.ElementCount)
		r.FullDecl = fmt.Sprintf("::std::array<%s, %v>", t.FullDecl, *val.ElementCount)
		r.LLDecl = fmt.Sprintf("::fidl::Array<%s, %v>", t.LLDecl, *val.ElementCount)
		r.LLFamily = Reference
		r.NeedsDtor = true
		r.Kind = ArrayKind
		r.ElementType = &t
		r.ElementCount = *val.ElementCount
	case fidl.VectorType:
		t := c.compileType(*val.ElementType)
		r.LLDecl = fmt.Sprintf("::fidl::VectorView<%s>", t.LLDecl)
		r.LLFamily = Vector
		if val.Nullable {
			r.Decl = fmt.Sprintf("::fidl::VectorPtr<%s>", t.Decl)
			r.FullDecl = fmt.Sprintf("::fidl::VectorPtr<%s>", t.FullDecl)
		} else {
			r.Decl = fmt.Sprintf("::std::vector<%s>", t.Decl)
			r.FullDecl = fmt.Sprintf("::std::vector<%s>", t.FullDecl)
		}
		r.NeedsDtor = true
		r.Kind = VectorKind
		r.ElementType = &t
	case fidl.StringType:
		r.LLDecl = "::fidl::StringView"
		r.LLFamily = String
		if val.Nullable {
			r.Decl = "::fidl::StringPtr"
		} else {
			r.Decl = "::std::string"
		}
		r.FullDecl = r.Decl
		r.NeedsDtor = true
		r.Kind = StringKind
	case fidl.HandleType:
		c.handleTypes[val.HandleSubtype] = struct{}{}
		r.Decl = fmt.Sprintf("::zx::%s", val.HandleSubtype)
		r.FullDecl = r.Decl
		r.LLDecl = r.Decl
		r.LLFamily = Reference
		r.NeedsDtor = true
		r.Kind = HandleKind
	case fidl.RequestType:
		r.Decl = fmt.Sprintf("::fidl::InterfaceRequest<%s>",
			c.compileCompoundIdentifier(val.RequestSubtype, "", "", false))
		r.FullDecl = fmt.Sprintf("::fidl::InterfaceRequest<%s>",
			c.compileCompoundIdentifier(val.RequestSubtype, "", "", true))
		r.LLDecl = "::zx::channel"
		r.LLFamily = Reference
		r.NeedsDtor = true
		r.Kind = RequestKind
	case fidl.PrimitiveType:
		r.Decl = c.compilePrimitiveSubtype(val.PrimitiveSubtype)
		r.FullDecl = r.Decl
		r.LLDecl = r.Decl
		r.LLFamily = TrivialCopy
		r.Kind = PrimitiveKind
	case fidl.IdentifierType:
		t := c.compileCompoundIdentifier(val.Identifier, "", "", false)
		ft := c.compileCompoundIdentifier(val.Identifier, "", "", true)
		declInfo, ok := c.decls[val.Identifier]
		if !ok {
			panic(fmt.Sprintf("unknown identifier: %v", val.Identifier))
		}
		declType := declInfo.Type
		if declType == fidl.ProtocolDeclType {
			r.Decl = fmt.Sprintf("::fidl::InterfaceHandle<class %s>", t)
			r.FullDecl = fmt.Sprintf("::fidl::InterfaceHandle<class %s>", ft)
			r.LLDecl = "::zx::channel"
			r.LLFamily = Reference
			r.NeedsDtor = true
			r.Kind = ProtocolKind
		} else {
			switch declType {
			case fidl.BitsDeclType:
				r.Kind = BitsKind
				r.LLFamily = TrivialCopy
			case fidl.EnumDeclType:
				r.Kind = EnumKind
				r.LLFamily = TrivialCopy
			case fidl.ConstDeclType:
				r.Kind = ConstKind
				r.LLFamily = Reference
			case fidl.StructDeclType:
				r.Kind = StructKind
				r.DeclarationName = val.Identifier
				r.LLFamily = Reference
				r.LLClass = ft
				r.LLPointer = val.Nullable
			case fidl.TableDeclType:
				r.Kind = TableKind
				r.DeclarationName = val.Identifier
				r.LLFamily = Reference
				r.LLClass = ft
				r.LLPointer = val.Nullable
			case fidl.UnionDeclType:
				r.Kind = UnionKind
				r.DeclarationName = val.Identifier
				r.LLFamily = Reference
				r.LLClass = ft
			default:
				panic(fmt.Sprintf("unknown declaration type: %v", declType))
			}

			if val.Nullable {
				r.Decl = fmt.Sprintf("::std::unique_ptr<%s>", t)
				r.FullDecl = fmt.Sprintf("::std::unique_ptr<%s>", ft)
				if declType == fidl.UnionDeclType {
					r.LLDecl = fmt.Sprintf("%s", ft)
				} else {
					r.LLDecl = fmt.Sprintf("::fidl::tracking_ptr<%s>", ft)
				}
				r.NeedsDtor = true
			} else {
				r.Decl = t
				r.FullDecl = ft
				r.LLDecl = ft
				r.NeedsDtor = true
			}
		}
	default:
		panic(fmt.Sprintf("unknown type kind: %v", val.Kind))
	}
	return r
}

func (c *compiler) compileBits(val fidl.Bits, appendNamespace string) Bits {
	r := Bits{
		Bits:      val,
		Namespace: c.namespace,
		Type:      c.compileType(val.Type).Decl,
		Name:      c.compileCompoundIdentifier(val.Name, "", appendNamespace, false),
		Mask:      val.Mask,
		MaskName:  c.compileCompoundIdentifier(val.Name, "Mask", appendNamespace, false),
	}
	for _, v := range val.Members {
		r.Members = append(r.Members, BitsMember{
			v.Attributes,
			changeIfReserved(v.Name, ""),
			c.compileConstant(v.Value, nil, val.Type, appendNamespace),
		})
	}
	return r
}

func (c *compiler) compileConst(val fidl.Const, appendNamespace string) Const {
	if val.Type.Kind == fidl.StringType {
		return Const{
			Attributes: val.Attributes,
			Extern:     true,
			Decorator:  "const",
			Type: Type{
				Decl:   "char",
				LLDecl: "char",
			},
			Name:  c.compileCompoundIdentifier(val.Name, "[]", appendNamespace, false),
			Value: c.compileConstant(val.Value, nil, val.Type, appendNamespace),
		}
	} else {
		t := c.compileType(val.Type)
		return Const{
			Attributes: val.Attributes,
			Extern:     false,
			Decorator:  "constexpr",
			Type:       t,
			Name:       c.compileCompoundIdentifier(val.Name, "", appendNamespace, false),
			Value:      c.compileConstant(val.Value, &t, val.Type, appendNamespace),
		}
	}
}

func (c *compiler) compileEnum(val fidl.Enum, appendNamespace string) Enum {
	r := Enum{
		Enum:      val,
		Namespace: c.namespace,
		Type:      c.compilePrimitiveSubtype(val.Type),
		Name:      c.compileCompoundIdentifier(val.Name, "", appendNamespace, false),
	}
	for _, v := range val.Members {
		r.Members = append(r.Members, EnumMember{
			EnumMember: v,
			Name:       changeIfReserved(v.Name, ""),
			// TODO(fxbug.dev/7660): When we expose types consistently in the IR, we
			// will not need to plug this here.
			Value: c.compileConstant(v.Value, nil, fidl.Type{
				Kind:             fidl.PrimitiveType,
				PrimitiveSubtype: val.Type,
			}, appendNamespace),
		})
	}
	return r
}

func (c *compiler) compileParameterArray(val []fidl.Parameter) []Parameter {
	var params []Parameter = []Parameter{}
	for _, v := range val {
		params = append(params, Parameter{
			Type:   c.compileType(v.Type),
			Name:   changeIfReserved(v.Name, ""),
			Offset: v.FieldShapeV1.Offset,
		})
	}
	return params
}

// LLContext indicates where the request/response is used.
// The allocation strategies differ for client and server contexts.
type LLContext int

const (
	clientContext LLContext = iota
	serverContext LLContext = iota
)

func (m Method) NewLLContextProps(context LLContext) LLContextProps {
	stackAllocRequest := false
	stackAllocResponse := false
	if context == clientContext {
		stackAllocRequest = len(m.Request) == 0 || (m.RequestSize+m.RequestMaxOutOfLine) < llcppMaxStackAllocSize
		stackAllocResponse = len(m.Response) == 0 || (!m.ResponseFlexible && (m.ResponseSize+m.ResponseMaxOutOfLine) < llcppMaxStackAllocSize)
	} else {
		stackAllocRequest = len(m.Request) == 0 || (!m.RequestFlexible && (m.RequestSize+m.RequestMaxOutOfLine) < llcppMaxStackAllocSize)
		stackAllocResponse = len(m.Response) == 0 || (m.ResponseSize+m.ResponseMaxOutOfLine) < llcppMaxStackAllocSize
	}

	stackUseRequest := 0
	stackUseResponse := 0
	if stackAllocRequest {
		stackUseRequest = m.RequestSize + m.RequestMaxOutOfLine
	}
	if stackAllocResponse {
		stackUseResponse = m.ResponseSize + m.ResponseMaxOutOfLine
	}
	return LLContextProps{
		StackAllocRequest:  stackAllocRequest,
		StackAllocResponse: stackAllocResponse,
		StackUseRequest:    stackUseRequest,
		StackUseResponse:   stackUseResponse,
	}
}

func (m Method) NewLLProps(protocolName string, reqTypeShape fidl.TypeShape, respTypeShape fidl.TypeShape) LLProps {
	return LLProps{
		ProtocolName:      protocolName,
		LinearizeRequest:  len(m.Request) > 0 && reqTypeShape.Depth > 0,
		LinearizeResponse: len(m.Response) > 0 && respTypeShape.Depth > 0,
		ClientContext:     m.NewLLContextProps(clientContext),
		ServerContext:     m.NewLLContextProps(serverContext),
	}
}

func (c *compiler) compileProtocol(val fidl.Protocol) *Protocol {
	protocolName := c.compileCompoundIdentifier(val.Name, "", "", false)
	methods := []Method{}
	for _, v := range val.Methods {
		name := changeIfReserved(v.Name, "")
		callbackType := ""
		if v.HasResponse {
			callbackType = changeIfReserved(v.Name, "Callback")
		}
		responseTypeNameSuffix := "ResponseTable"
		if !v.HasRequest {
			responseTypeNameSuffix = "EventTable"
		}

		var result *Result
		if v.HasResponse && len(v.Response) == 1 && v.Response[0].Name == "result" {
			result = c.resultForUnion[v.Response[0].Type.Identifier]
		}

		var computedResponseReceivedMaxSize int
		if v.ResponseTypeShapeV1.HasFlexibleEnvelope {
			computedResponseReceivedMaxSize = (1 << 32) - 1
		} else {
			computedResponseReceivedMaxSize = v.ResponseTypeShapeV1.InlineSize + v.ResponseTypeShapeV1.MaxOutOfLine
		}

		m := Method{
			Attributes:              v.Attributes,
			Name:                    name,
			NameInLowerSnakeCase:    fidl.ToSnakeCase(name),
			Ordinal:                 v.Ordinal,
			OrdinalName:             fmt.Sprintf("k%s_%s_Ordinal", protocolName, v.Name),
			HasRequest:              v.HasRequest,
			Request:                 c.compileParameterArray(v.Request),
			RequestSize:             v.RequestTypeShapeV1.InlineSize,
			RequestTypeName:         fmt.Sprintf("%s_%s%sRequestTable", c.symbolPrefix, protocolName, v.Name),
			RequestMaxHandles:       v.RequestTypeShapeV1.MaxHandles,
			RequestMaxOutOfLine:     v.RequestTypeShapeV1.MaxOutOfLine,
			RequestSentMaxSize:      v.RequestTypeShapeV1.InlineSize + v.RequestTypeShapeV1.MaxOutOfLine,
			RequestPadding:          v.RequestTypeShapeV1.HasPadding,
			RequestFlexible:         v.RequestTypeShapeV1.HasFlexibleEnvelope,
			RequestHasPointer:       v.RequestTypeShapeV1.Depth > 0,
			RequestIsResource:       v.RequestTypeShapeV1.IsResource,
			HasResponse:             v.HasResponse,
			Response:                c.compileParameterArray(v.Response),
			ResponseSize:            v.ResponseTypeShapeV1.InlineSize,
			ResponseTypeName:        fmt.Sprintf("%s_%s%s%s", c.symbolPrefix, protocolName, v.Name, responseTypeNameSuffix),
			ResponseMaxHandles:      v.ResponseTypeShapeV1.MaxHandles,
			ResponseMaxOutOfLine:    v.ResponseTypeShapeV1.MaxOutOfLine,
			ResponseSentMaxSize:     v.ResponseTypeShapeV1.InlineSize + v.ResponseTypeShapeV1.MaxOutOfLine,
			ResponseReceivedMaxSize: computedResponseReceivedMaxSize,
			ResponsePadding:         v.ResponseTypeShapeV1.HasPadding,
			ResponseFlexible:        v.ResponseTypeShapeV1.HasFlexibleEnvelope,
			ResponseHasPointer:      v.ResponseTypeShapeV1.Depth > 0,
			ResponseIsResource:      v.ResponseTypeShapeV1.IsResource,
			CallbackType:            callbackType,
			ResponseHandlerType:     fmt.Sprintf("%s_%s_ResponseHandler", protocolName, v.Name),
			ResponderType:           fmt.Sprintf("%s_%s_Responder", protocolName, v.Name),
			Transitional:            v.IsTransitional(),
			Result:                  result,
		}

		m.LLProps = m.NewLLProps(protocolName, v.RequestTypeShapeV1, v.ResponseTypeShapeV1)
		methods = append(methods, m)
	}

	r := protocolInner{
		Attributes:          val.Attributes,
		Namespace:           c.namespace,
		Name:                protocolName,
		ClassName:           c.compileCompoundIdentifier(val.Name, "_clazz", "", false),
		ServiceName:         val.GetServiceName(),
		ProxyName:           c.compileCompoundIdentifier(val.Name, "_Proxy", "", false),
		StubName:            c.compileCompoundIdentifier(val.Name, "_Stub", "", false),
		EventSenderName:     c.compileCompoundIdentifier(val.Name, "_EventSender", "", false),
		SyncName:            c.compileCompoundIdentifier(val.Name, "_Sync", "", false),
		SyncProxyName:       c.compileCompoundIdentifier(val.Name, "_SyncProxy", "", false),
		RequestEncoderName:  c.compileCompoundIdentifier(val.Name, "_RequestEncoder", "", false),
		RequestDecoderName:  c.compileCompoundIdentifier(val.Name, "_RequestDecoder", "", false),
		ResponseEncoderName: c.compileCompoundIdentifier(val.Name, "_ResponseEncoder", "", false),
		ResponseDecoderName: c.compileCompoundIdentifier(val.Name, "_ResponseDecoder", "", false),
		Methods:             methods,
	}.build()
	return r
}

func (c *compiler) compileService(val fidl.Service) Service {
	s := Service{
		Attributes:  val.Attributes,
		Namespace:   c.namespace,
		Name:        c.compileCompoundIdentifier(val.Name, "", "", false),
		ServiceName: val.GetServiceName(),
	}

	for _, v := range val.Members {
		s.Members = append(s.Members, c.compileServiceMember(v))
	}
	return s
}

func (c *compiler) compileServiceMember(val fidl.ServiceMember) ServiceMember {
	return ServiceMember{
		Attributes:   val.Attributes,
		ProtocolType: c.compileCompoundIdentifier(val.Type.Identifier, "", "", false),
		Name:         string(val.Name),
		MethodName:   changeIfReserved(val.Name, ""),
	}
}

func (c *compiler) compileStructMember(val fidl.StructMember, appendNamespace string) StructMember {
	t := c.compileType(val.Type)

	defaultValue := ""
	if val.MaybeDefaultValue != nil {
		defaultValue = c.compileConstant(*val.MaybeDefaultValue, &t, val.Type, appendNamespace)
	}

	return StructMember{
		Attributes:   val.Attributes,
		Type:         t,
		Name:         changeIfReserved(val.Name, ""),
		DefaultValue: defaultValue,
		Offset:       val.FieldShapeV1.Offset,
	}
}

func (c *compiler) compileStruct(val fidl.Struct, appendNamespace string) Struct {
	name := c.compileCompoundIdentifier(val.Name, "", appendNamespace, false)
	tableType := c.compileTableType(val.Name)
	r := Struct{
		Struct:       val,
		Namespace:    c.namespace,
		Name:         name,
		TableType:    tableType,
		Members:      []StructMember{},
		InlineSize:   val.TypeShapeV1.InlineSize,
		MaxHandles:   val.TypeShapeV1.MaxHandles,
		MaxOutOfLine: val.TypeShapeV1.MaxOutOfLine,
		MaxSentSize:  val.TypeShapeV1.InlineSize + val.TypeShapeV1.MaxOutOfLine,
		HasPadding:   val.TypeShapeV1.HasPadding,
		HasPointer:   val.TypeShapeV1.Depth > 0,
		IsResource:   val.TypeShapeV1.IsResource,
	}

	for _, v := range val.Members {
		r.Members = append(r.Members, c.compileStructMember(v, appendNamespace))
	}

	result := c.resultForStruct[val.Name]
	if result != nil {
		(*result).ValueMembers = r.Members
		memberTypeDecls := []string{}
		for _, m := range r.Members {
			memberTypeDecls = append(memberTypeDecls, m.Type.Decl)
		}
		(*result).ValueTupleDecl = fmt.Sprintf("std::tuple<%s>", strings.Join(memberTypeDecls, ", "))

		if len(r.Members) == 0 {
			(*result).ValueDecl = "void"
		} else if len(r.Members) == 1 {
			(*result).ValueDecl = r.Members[0].Type.Decl
		} else {

			(*result).ValueDecl = (*result).ValueTupleDecl
		}

		r.IsResultValue = true
		r.Result = result
	}

	if len(r.Members) == 0 {
		r.Members = []StructMember{
			c.compileStructMember(fidl.EmptyStructMember("__reserved"), appendNamespace),
		}
	}

	// Construct a deduped list of decls for IsMemcpyCompatible template definitions.
	memcpyCompatibleDepMap := make(map[string]struct{})
	for _, member := range r.Members {
		// The dangerous identifiers test package contains identifiers that won't compile.
		// e.g. ::fidl::test::dangerous::struct::types::camel::Interface gives an
		// "expected unqualified-id" error because of "struct".
		// There isn't an easily accessible dangerous identifiers list to replace identifiers.
		if strings.Contains(member.Type.FullDecl, "::fidl::test::dangerous::") {
			memcpyCompatibleDepMap = nil
			break
		}
		memcpyCompatibleDepMap[member.Type.FullDecl] = struct{}{}
	}
	for decl := range memcpyCompatibleDepMap {
		r.FullDeclMemcpyCompatibleDeps = append(r.FullDeclMemcpyCompatibleDeps, decl)
	}
	sort.Strings(r.FullDeclMemcpyCompatibleDeps)

	return r
}

func (c *compiler) compileTableMember(val fidl.TableMember, appendNamespace string, index int) TableMember {
	t := c.compileType(val.Type)

	defaultValue := ""
	if val.MaybeDefaultValue != nil {
		defaultValue = c.compileConstant(*val.MaybeDefaultValue, &t, val.Type, appendNamespace)
	}

	return TableMember{
		Attributes:         val.Attributes,
		Type:               t,
		Name:               changeIfReserved(val.Name, ""),
		DefaultValue:       defaultValue,
		Ordinal:            val.Ordinal,
		FieldPresenceIsSet: fmt.Sprintf("field_presence_.IsSet<%d>()", val.Ordinal-1),
		FieldPresenceSet:   fmt.Sprintf("field_presence_.Set<%d>()", val.Ordinal-1),
		FieldPresenceClear: fmt.Sprintf("field_presence_.Clear<%d>()", val.Ordinal-1),
		FieldDataName:      fmt.Sprintf("%s_value_", val.Name),
		MethodHasName:      fmt.Sprintf("has_%s", val.Name),
		MethodClearName:    fmt.Sprintf("clear_%s", val.Name),
		ValueUnionName:     fmt.Sprintf("ValueUnion_%s", val.Name),
	}
}

func (c *compiler) compileTable(val fidl.Table, appendNamespace string) Table {
	name := c.compileCompoundIdentifier(val.Name, "", appendNamespace, false)
	tableType := c.compileTableType(val.Name)
	r := Table{
		Table:          val,
		Namespace:      c.namespace,
		Name:           name,
		TableType:      tableType,
		Members:        nil,
		InlineSize:     val.TypeShapeV1.InlineSize,
		BiggestOrdinal: 0,
		MaxHandles:     val.TypeShapeV1.MaxHandles,
		MaxOutOfLine:   val.TypeShapeV1.MaxOutOfLine,
		MaxSentSize:    val.TypeShapeV1.InlineSize + val.TypeShapeV1.MaxOutOfLine,
		HasPointer:     val.TypeShapeV1.Depth > 0,
		IsResource:     val.TypeShapeV1.IsResource,
	}

	for i, v := range val.SortedMembersNoReserved() {
		m := c.compileTableMember(v, appendNamespace, i)
		if m.Ordinal > r.BiggestOrdinal {
			r.BiggestOrdinal = m.Ordinal
		}
		r.Members = append(r.Members, m)
	}

	r.FrameItems = make([]TableFrameItem, r.BiggestOrdinal)
	for i := 0; i < len(r.FrameItems); i++ {
		r.FrameItems[i] = TableFrameItem{
			LLDecl: "void",
			Name:   fmt.Sprintf("reserved_%d", i),
		}
	}
	for _, member := range r.Members {
		r.FrameItems[member.Ordinal-1] = TableFrameItem{
			LLDecl: member.Type.LLDecl,
			Name:   member.Name,
		}
	}

	return r
}

func (c *compiler) compileUnionMember(val fidl.UnionMember) UnionMember {
	n := changeIfReserved(val.Name, "")
	return UnionMember{
		Attributes:  val.Attributes,
		Ordinal:     uint64(val.Ordinal),
		Type:        c.compileType(val.Type),
		Name:        n,
		StorageName: changeIfReserved(val.Name, "_"),
		TagName:     fmt.Sprintf("k%s", fidl.ToUpperCamelCase(n)),
		Offset:      val.Offset,
	}
}

func (c *compiler) compileUnion(val fidl.Union) Union {
	name := c.compileCompoundIdentifier(val.Name, "", "", false)
	tableType := c.compileTableType(val.Name)
	r := Union{
		Union:        val,
		Namespace:    c.namespace,
		Name:         name,
		TableType:    tableType,
		InlineSize:   val.TypeShapeV1.InlineSize,
		MaxHandles:   val.TypeShapeV1.MaxHandles,
		MaxOutOfLine: val.TypeShapeV1.MaxOutOfLine,
		HasPointer:   val.TypeShapeV1.Depth > 0,
		IsResource:   val.TypeShapeV1.IsResource,
	}

	for _, v := range val.Members {
		if v.Reserved {
			continue
		}
		r.Members = append(r.Members, c.compileUnionMember(v))
	}

	if val.Attributes.HasAttribute("Result") {
		if len(r.Members) != 2 {
			panic(fmt.Sprintf("result union must have two members: %v", val.Name))
		}
		if val.Members[0].Type.Kind != fidl.IdentifierType {
			panic(fmt.Sprintf("value member of result union must be an identifier: %v", val.Name))
		}
		valueStructDeclInfo, ok := c.decls[val.Members[0].Type.Identifier]
		if !ok {
			panic(fmt.Sprintf("unknown identifier: %v", val.Members[0].Type.Identifier))
		}
		if valueStructDeclInfo.Type != "struct" {
			panic(fmt.Sprintf("first member of result union not a struct: %v", val.Name))
		}
		result := Result{
			ResultDecl:      r.Name,
			ValueStructDecl: r.Members[0].Type.Decl,
			ErrorDecl:       r.Members[1].Type.Decl,
		}
		c.resultForStruct[val.Members[0].Type.Identifier] = &result
		c.resultForUnion[val.Name] = &result
		r.Result = &result
	}

	return r
}

func compile(r fidl.Root, namespaceFormatter func(fidl.LibraryIdentifier, string) string) Root {
	root := Root{}
	library := make(fidl.LibraryIdentifier, 0)
	raw_library := make(fidl.LibraryIdentifier, 0)
	for _, identifier := range fidl.ParseLibraryName(r.Name) {
		safe_name := changeIfReserved(identifier, "")
		library = append(library, fidl.Identifier(safe_name))
		raw_library = append(raw_library, identifier)
	}
	c := compiler{
		namespaceFormatter(library, ""),
		formatLibraryPrefix(raw_library),
		r.DeclsWithDependencies(),
		map[fidl.EncodedCompoundIdentifier]Decl{},
		fidl.ParseLibraryName(r.Name),
		make(map[fidl.HandleSubtype]struct{}),
		namespaceFormatter,
		make(map[fidl.EncodedCompoundIdentifier]*Result),
		make(map[fidl.EncodedCompoundIdentifier]*Result),
	}

	root.Library = library
	libraryReversed := make(fidl.LibraryIdentifier, len(library))
	for i, j := 0, len(library)-1; i < len(library); i, j = i+1, j-1 {
		libraryReversed[i] = library[j]
	}
	for i, identifier := range library {
		libraryReversed[len(libraryReversed)-i-1] = identifier
	}
	root.LibraryReversed = libraryReversed

	for _, v := range r.Bits {
		d := c.compileBits(v, "")
		c.declarations[v.Name] = &d
	}

	for _, v := range r.Consts {
		d := c.compileConst(v, "")
		c.declarations[v.Name] = &d
	}

	for _, v := range r.Enums {
		d := c.compileEnum(v, "")
		c.declarations[v.Name] = &d
	}

	// Note: for Result calculation unions must be compiled before structs.
	for _, v := range r.Unions {
		d := c.compileUnion(v)
		c.declarations[v.Name] = &d
	}

	for _, v := range r.Structs {
		// TODO(fxbug.dev/7704) remove once anonymous structs are supported
		if v.Anonymous {
			continue
		}
		d := c.compileStruct(v, "")
		c.declarations[v.Name] = &d
	}

	for _, v := range r.Tables {
		d := c.compileTable(v, "")
		c.declarations[v.Name] = &d
	}

	for _, v := range r.Protocols {
		d := c.compileProtocol(v)
		c.declarations[v.Name] = d
	}

	for _, v := range r.Services {
		d := c.compileService(v)
		c.declarations[v.Name] = &d
	}

	for _, v := range c.declarations {
		v.compileProperties(c)
	}

	for _, v := range r.DeclOrder {
		// We process only a subset of declarations mentioned in the declaration
		// order, ignore those we do not support.
		if d, known := c.declarations[v]; known {
			root.Decls = append(root.Decls, d)
		}
	}

	for _, l := range r.Libraries {
		if l.Name == r.Name {
			// We don't need to include our own header.
			continue
		}
		libraryIdent := fidl.ParseLibraryName(l.Name)
		root.Headers = append(root.Headers, formatLibraryPath(libraryIdent))
		root.FuzzerHeaders = append(root.FuzzerHeaders, formatLibraryPath(libraryIdent))
	}

	// zx::channel is always referenced by the protocols in llcpp bindings API
	if len(r.Protocols) > 0 {
		c.handleTypes["channel"] = struct{}{}
	}

	// find all unique handle types referenced by the library
	var handleTypes []string
	for k := range c.handleTypes {
		handleTypes = append(handleTypes, string(k))
	}
	sort.Sort(sort.StringSlice(handleTypes))
	root.HandleTypes = handleTypes

	return root
}

func CompileHL(r fidl.Root) Root {
	return compile(r.ForBindings("hlcpp"), formatNamespace)
}

func CompileLL(r fidl.Root) Root {
	return compile(r.ForBindings("llcpp"), formatLLNamespace)
}

func CompileLibFuzzer(r fidl.Root) Root {
	return compile(r.ForBindings("libfuzzer"), formatNamespace)
}
