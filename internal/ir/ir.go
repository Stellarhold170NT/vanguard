package ir

// Location identifies a source position. Every IR node carries one so any
// finding can point at file:line:col (charter §5.2, R1/UC1).
//
// File is relative to the scan root; Line and Column are 1-based (0 marks a
// synthetic position). The column encoding (bytes vs UTF-16 code units —
// SARIF columnKind, §6.8) is pinned by w2-02.
type Location struct {
	File   string
	Line   int
	Column int
}

// Node is any IR node the engine can visit. Checks (charter §5.4) receive
// nodes as Node and anchor their findings at Loc(). w2-02 makes every node
// in the inventory below implement it.
type Node interface {
	Loc() Location
}

// Verb is the HTTP verb of a Method, or RPC for a gRPC call declared in
// .proto (charter §5.2).
type Verb string

const (
	VerbGet    Verb = "GET"
	VerbPost   Verb = "POST"
	VerbPut    Verb = "PUT"
	VerbPatch  Verb = "PATCH"
	VerbDelete Verb = "DELETE"
	VerbRPC    Verb = "rpc"
)

// ParamIn is where a Method parameter is bound. "body" covers
// @RequestBody-style parameters: the rule matrix (§5.2, R2xx) needs body
// params visible both here and as Method.Payload — w2-02 pins how the two
// views stay consistent.
type ParamIn string

const (
	ParamInPath   ParamIn = "path"
	ParamInQuery  ParamIn = "query"
	ParamInHeader ParamIn = "header"
	ParamInBody   ParamIn = "body"
)

// TypeRef names a payload/response/field type the way the source does.
type TypeRef struct {
	Name         string // as written, e.g. "Page<YouthResponse>"
	Package      string // owning package/namespace, "" when unresolved
	IsCollection bool
}

// PaginationStyle enumerates the pagination shapes the IR distinguishes.
type PaginationStyle string

const (
	PaginationPageable PaginationStyle = "pageable" // Spring Pageable
	PaginationParams   PaginationStyle = "params"   // page/size query params
	PaginationNone     PaginationStyle = "none"
)

// Source describes the language and framework detected for a scan.
type Source struct {
	Lang             string // e.g. "java" (discovery.Adapter.Language)
	Framework        string // e.g. "spring-boot"
	FrameworkVersion string // e.g. "4.0.2"; "" when undetected
}

// Param is one declared parameter of a Method.
type Param struct {
	Name       string
	In         ParamIn
	Type       TypeRef
	Validation string // raw validation annotation, e.g. "@Max(100)"
	Location   Location
}

// Response describes the declared response shape of a Method.
type Response struct {
	Type         TypeRef
	StatusCode   int    // declared status; 0 = inferred from verb
	IsCollection bool   // List<...>, Page<...>, arrays
	Envelope     string // wrapper type name, "" when bare
	Location     Location
}

// Pagination captures how a Method pages its collection responses (R3xx).
type Pagination struct {
	Style          PaginationStyle
	PageSizeCapped bool // size/limit param has a @Max-style cap
}

// Method is one API operation: an annotated handler method (REST) or an rpc
// (gRPC). Path is the full path after merging the service base path
// (w3-02, tested for double-slash).
type Method struct {
	OperationName string // Java method name or rpc name
	Verb          Verb
	Path          string
	Params        []Param
	Payload       *TypeRef // request body type; nil when none
	Response      Response
	Pagination    Pagination
	Annotations   map[string]string // raw annotation text by simple name
	Location      Location
}

// Service is one REST controller class or one .proto service declaration.
type Service struct {
	Name        string
	BasePath    string // merged, normalized (w3-02)
	Annotations map[string]string
	Methods     []Method
	Location    Location
}

// TypeKind classifies a Type (charter §5.2: record|pojo).
type TypeKind string

const (
	KindRecord TypeKind = "record"
	KindPOJO   TypeKind = "pojo"
)

// Field is one field of a Type.
type Field struct {
	Name        string
	JSONName    string // serialized name; "" when same as Name
	Type        TypeRef
	Annotations map[string]string
	Location    Location
}

// Type is a DTO/POJO/record inside scan scope (input of the R4xx family).
type Type struct {
	Name     string
	Kind     TypeKind
	Package  string
	IsEntity bool // persistence annotation seen (R4xx-01 signal)
	Fields   []Field
	Location Location
}

// ErrorHandler is one global exception mapping (from @ControllerAdvice /
// @ExceptionHandler, w3-02; input of the R5xx family).
type ErrorHandler struct {
	ExceptionType string
	ResponseType  string
	StatusCode    int
	Location      Location
}

// ErrorScheme collects the global error-envelope handlers of the scan.
type ErrorScheme struct {
	Handlers []ErrorHandler
}

// GrpcService is a .proto service read at declaration level only (w3-06;
// input of the R6xx family).
type GrpcService struct {
	Name     string
	Rpcs     []string
	Location Location
}

// Diagnostic is a non-fatal problem (parse error, unresolved annotation).
// Best-effort contract: diagnostics never abort a scan and never change the
// exit code (charter §5.3); renderers surface them in the summary and in
// verbose mode.
type Diagnostic struct {
	Message  string
	Location Location
}

// ApiSurface is the root of the IR: everything an adapter produced, and
// everything the engine and renderers consume.
type ApiSurface struct {
	Source       Source
	Services     []Service
	Types        []Type
	ErrorScheme  ErrorScheme
	GrpcServices []GrpcService
	Diagnostics  []Diagnostic
}
