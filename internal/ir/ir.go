package ir

// Location identifies a source position. Every IR node carries one so any
// finding can point at file:line:col (charter §5.2, R1/UC1).
//
// File is relative to the scan root, uses "/" separators and never contains
// "..". Line and Column are 1-based; Column counts UTF-16 code units — the
// unit the SARIF renderer declares as columnKind "utf16CodeUnits"
// (charter §6.8). For ASCII source a column equals a byte offset; a
// non-BMP character counts as 2. Line == 0 && Column == 0 marks a synthetic
// position (the adapter synthesized the node, it has no source position);
// every other combination must be fully positive — Validate enforces this.
type Location struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// Node is any IR node the engine can visit. Checks (charter §5.4) receive
// nodes as Node and anchor their findings at Loc(). Every inventory struct
// that carries a Location implements Node; the pure containers (ApiSurface,
// ErrorScheme, Source, TypeRef) and Diagnostic do not — rules never run on
// diagnostics, renderers surface them instead.
type Node interface {
	Loc() Location
}

// Verb is the HTTP verb of a Method, or RPC for a gRPC call. The spelling
// of every constant is locked by the charter §5.2 inventory and by the
// golden snapshots: the five HTTP verbs are upper-case, as they appear on
// the wire; rpc is lower-case, verbatim from the charter inventory — it
// marks a gRPC operation, where HTTP verb semantics do not apply.
type Verb string

const (
	VerbGet    Verb = "GET"
	VerbPost   Verb = "POST"
	VerbPut    Verb = "PUT"
	VerbPatch  Verb = "PATCH"
	VerbDelete Verb = "DELETE"
	VerbRPC    Verb = "rpc"
)

// Valid reports whether v is one of the six declared verbs. Adapters must
// only emit declared verbs; Validate flags anything else.
func (v Verb) Valid() bool {
	switch v {
	case VerbGet, VerbPost, VerbPut, VerbPatch, VerbDelete, VerbRPC:
		return true
	default:
		return false
	}
}

// IsHTTP reports whether v is an HTTP verb (valid and not rpc). The R2xx
// family (verb/body rules) selects on this — a rule that reasons about
// request bodies must not fire on gRPC operations.
func (v Verb) IsHTTP() bool {
	return v.Valid() && v != VerbRPC
}

// ParamIn is where a Method parameter is bound, mirroring Spring's binding
// annotations: path = @PathVariable (a path template segment), query =
// @RequestParam, header = @RequestHeader, body = @RequestBody (the request
// body itself). gRPC operations carry no parameters: the whole request
// message is Payload instead.
//
// Invariant (this pins the w2-01 hand-over note): for HTTP verbs, Payload
// and the in=body Param are two views of the same @RequestBody parameter —
// adapters fill both, Validate enforces that they agree.
type ParamIn string

const (
	ParamInPath   ParamIn = "path"
	ParamInQuery  ParamIn = "query"
	ParamInHeader ParamIn = "header"
	ParamInBody   ParamIn = "body"
)

// TypeRef names a type the way the source writes it. It is a value, not a
// node: the Param, Field, Response or Payload holding it carries the
// Location.
//
// Name is the source spelling including generics, e.g. "Page<YouthResponse>"
// or "List<String>". Package is the fully qualified package of the base
// type, "" when unresolved (a primitive, a java.lang type, or an import the
// adapter could not chase). IsCollection marks wrapper erasure: List<...>,
// Page<...>, arrays and varargs all count as collections (R3xx reads it).
type TypeRef struct {
	Name         string `json:"name"`
	Package      string `json:"package,omitempty"`
	IsCollection bool   `json:"isCollection,omitempty"`
}

// Payload is the request-body view of TypeRef: the @RequestBody parameter
// type of a Spring handler, or the request message type of a proto rpc.
// Method.Payload points at it (nil = the operation takes no body). It is an
// alias, so adapters and rules can spell the concept they mean.
type Payload = TypeRef

// Source describes the language and framework a scan detected. It comes
// from discovery (w2-03): Lang is the Adapter.Language() id (e.g. "java"),
// Framework the normalized framework id (e.g. "spring-boot"), and
// FrameworkVersion the detected version (e.g. "4.0.2", "" when undetected).
// The engine copies it into every Report and renderers print it.
type Source struct {
	Lang             string `json:"lang"`
	Framework        string `json:"framework"`
	FrameworkVersion string `json:"frameworkVersion,omitempty"`
}

// Param is one declared parameter of a Method: a @PathVariable,
// @RequestParam, @RequestHeader or @RequestBody parameter in Spring MVC.
//
// Name is the bind name as written in the annotation. Validation carries
// the raw constraint annotation text (e.g. "@Max(100)", "" when
// unconstrained) — the R3xx family reads it to detect a page-size cap.
type Param struct {
	Name       string   `json:"name"`
	In         ParamIn  `json:"in"`
	Type       TypeRef  `json:"type"`
	Validation string   `json:"validation,omitempty"`
	Location   Location `json:"location"`
}

// Loc implements Node.
func (p Param) Loc() Location { return p.Location }

// Response describes the declared response shape of a Method.
//
// Type is the response body type with transport wrappers removed (the
// adapter unwraps ResponseEntity/Mono/Flux; container shapes such as
// List<...> or Page<...> stay in the spelling and set IsCollection). It may
// be the zero TypeRef for handlers that return nothing (void).
// StatusCode is the declared HTTP status — from @ResponseStatus or the
// ResponseEntity status argument; 0 means "not declared", and consumers
// infer it from the verb by convention. Envelope is the wrapper type name
// when the body is wrapped in an envelope (e.g. ApiResponse{data, total}),
// "" when the body is the bare type.
type Response struct {
	Type         TypeRef  `json:"type"`
	StatusCode   int      `json:"statusCode,omitempty"`
	IsCollection bool     `json:"isCollection,omitempty"`
	Envelope     string   `json:"envelope,omitempty"`
	Location     Location `json:"location"`
}

// Loc implements Node.
func (r Response) Loc() Location { return r.Location }

// PaginationStyle enumerates the pagination shapes the IR distinguishes
// (charter §5.2: pageable|params|none).
type PaginationStyle string

const (
	PaginationPageable PaginationStyle = "pageable" // Spring Pageable argument
	PaginationParams   PaginationStyle = "params"   // page/size query parameters
	PaginationNone     PaginationStyle = "none"     // not paginated
)

// Valid reports whether s is one of the declared styles. Adapters must
// state the shape explicitly (also "none" for plain methods); Validate
// flags anything else.
func (s PaginationStyle) Valid() bool {
	switch s {
	case PaginationPageable, PaginationParams, PaginationNone:
		return true
	default:
		return false
	}
}

// Pagination captures how a Method pages its collection responses — the
// input of the R3xx family.
//
// Style is pageable when the handler takes a Spring Pageable argument,
// params when it takes page/size(-like) query parameters, and none
// otherwise. PageSizeCapped says the size/limit parameter carries a
// @Max-style constraint (R3xx-02). The struct is derived from the method
// signature; Location points at the signal (the Pageable parameter or the
// page/size parameters), falling back to the Method's position.
type Pagination struct {
	Style          PaginationStyle `json:"style"`
	PageSizeCapped bool            `json:"pageSizeCapped,omitempty"`
	Location       Location        `json:"location"`
}

// Loc implements Node.
func (p Pagination) Loc() Location { return p.Location }

// Method is one API operation: a Spring handler method (its mapping
// annotation decides the Verb) or a proto rpc.
//
// Containment invariant: a Method exists only as an element of
// Service.Methods — the IR has no standalone methods, so every Method
// always belongs to exactly one Service (the Resource of the REST
// reading). Rules that need the owning Service (R1xx and R6xx need
// BasePath) get it from the engine context (w2-04), which walks the tree.
//
// OperationName is the Java method name or the rpc name. Path is the FULL
// request path after the adapter merged the service base path (w3-02 owns
// the merge and its double-slash tests); it is a normalized absolute path
// starting with "/". Payload is the request body (nil = none; for rpc the
// request message type). Annotations follow the shared convention: key =
// annotation simple name (e.g. "ResponseStatus"), value = raw source text
// (e.g. "@ResponseStatus(HttpStatus.NOT_FOUND)"); when the same simple
// name appears twice the adapter joins the raw texts with "\n"; map keys
// serialize sorted (B-6).
type Method struct {
	OperationName string            `json:"operationName"`
	Verb          Verb              `json:"verb"`
	Path          string            `json:"path"`
	Params        []Param           `json:"params,omitempty"`
	Payload       *TypeRef          `json:"payload,omitempty"`
	Response      Response          `json:"response"`
	Pagination    Pagination        `json:"pagination"`
	Annotations   map[string]string `json:"annotations,omitempty"`
	Location      Location          `json:"location"`
}

// Loc implements Node.
func (m Method) Loc() Location { return m.Location }

// Service is one grouping node of the API surface: a Spring controller
// class (@RestController — the Resource of the REST reading) or one
// service declaration of a .proto file (gRPC).
//
// Name is the class or service simple name. BasePath is the merged,
// normalized base path: the class-level @RequestMapping value for REST
// ("" when the controller has none), or the proto package path for gRPC —
// w3-02 owns normalization and its tests. Annotations follow the shared
// convention described on Method.
type Service struct {
	Name        string            `json:"name"`
	BasePath    string            `json:"basePath,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Methods     []Method          `json:"methods,omitempty"`
	Location    Location          `json:"location"`
}

// Loc implements Node.
func (s Service) Loc() Location { return s.Location }

// Resource is the REST-world reading of Service: in an HTTP API the unit
// the rules talk about is a resource — in Spring, a @RestController class.
// The charter names the node Service so the same node also covers proto
// services; the alias lets adapters and rules use whichever reading fits.
type Resource = Service

// TypeKind classifies a Type (charter §5.2: record|pojo).
type TypeKind string

const (
	KindRecord TypeKind = "record" // Java record / immutable data carrier
	KindPOJO   TypeKind = "pojo"   // classic class with accessors
)

// Field is one field of a Type.
//
// Name is the source field name; JSONName is the serialized name — from
// @JsonProperty/@SerializedName or the accessor convention — and "" means
// "same as Name" (R4xx reads this pair to catch naming mismatches).
// Annotations follow the shared convention described on Method.
type Field struct {
	Name        string            `json:"name"`
	JSONName    string            `json:"jsonName,omitempty"`
	Type        TypeRef           `json:"type"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Location    Location          `json:"location"`
}

// Loc implements Node.
func (f Field) Loc() Location { return f.Location }

// Type is a DTO, record or entity class inside the scan scope — the input
// of the R4xx family.
//
// Name is the class simple name, Package its fully qualified package.
// Kind is record or pojo. IsEntity marks a persistence-annotated class
// (@Entity/@Document — the R4xx-01 signal that an entity is exposed
// directly). Fields lists the class fields in declaration order.
type Type struct {
	Name     string   `json:"name"`
	Kind     TypeKind `json:"kind"`
	Package  string   `json:"package,omitempty"`
	IsEntity bool     `json:"isEntity,omitempty"`
	Fields   []Field  `json:"fields,omitempty"`
	Location Location `json:"location"`
}

// Loc implements Node.
func (t Type) Loc() Location { return t.Location }

// ErrorHandler is one global exception mapping — one @ExceptionHandler
// method of a @ControllerAdvice class (w3-02 collects them; the R5xx
// family reads them to check response consistency).
//
// ExceptionType is the exception class as written in the annotation
// (usually the simple name). ResponseType is the body type the handler
// responds with ("" when it returns void). StatusCode is the declared
// status — from @ResponseStatus; 0 means undeclared, and consumers infer
// 4xx/5xx from the exception.
type ErrorHandler struct {
	ExceptionType string   `json:"exceptionType"`
	ResponseType  string   `json:"responseType,omitempty"`
	StatusCode    int      `json:"statusCode,omitempty"`
	Location      Location `json:"location"`
}

// Loc implements Node.
func (e ErrorHandler) Loc() Location { return e.Location }

// ErrorScheme collects the global error-envelope handlers of a scan. It is
// a pure container: it carries no Location of its own because a scan may
// read several advice classes — every Handler carries its own.
type ErrorScheme struct {
	Handlers []ErrorHandler `json:"handlers,omitempty"`
}

// GrpcService is a .proto service read at declaration level only (w3-06):
// the service Name and the names of its Rpcs in declaration order. The
// R6xx family uses it to check naming and versioning; message bodies and
// fields are out of scope for the declaration-level read.
type GrpcService struct {
	Name     string   `json:"name"`
	Rpcs     []string `json:"rpcs,omitempty"`
	Location Location `json:"location"`
}

// Loc implements Node.
func (g GrpcService) Loc() Location { return g.Location }

// Diagnostic is a non-fatal problem recorded during parsing: a file that
// could not be parsed, an annotation that could not be resolved. The
// best-effort contract (charter §5.3) says diagnostics never abort a scan
// and never change the exit code; renderers surface them in the summary
// and in verbose mode. Diagnostics are not Node: rules never run on them.
type Diagnostic struct {
	Message  string   `json:"message"` // one English sentence, reason first
	Location Location `json:"location"`
}

// ApiSurface is the root of the IR: everything an adapter produced and
// everything the engine and renderers consume. Adapter.Parse returns it;
// Engine.Run walks it; the engine copies Source and Diagnostics into every
// Report.
//
// Determinism contract (B-6): every slice is in source order and every map
// serializes with sorted keys, so marshalling the same scan twice — and
// across runs — yields byte-identical JSON (golden snapshots, w4-03).
// Absent collections marshal as absent, never null: adapters may use nil
// and empty slices/maps interchangeably.
//
// The IR is pure Go: this package imports nothing beyond the standard
// library and knows nothing about tree-sitter, Java or any language.
type ApiSurface struct {
	Source       Source        `json:"source"`
	Services     []Service     `json:"services,omitempty"`
	Types        []Type        `json:"types,omitempty"`
	ErrorScheme  ErrorScheme   `json:"errorScheme"`
	GrpcServices []GrpcService `json:"grpcServices,omitempty"`
	Diagnostics  []Diagnostic  `json:"diagnostics,omitempty"`
}
