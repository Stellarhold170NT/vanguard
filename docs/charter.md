# Vanguard Charter v1

**Task**: w1-04 · **Role**: architect · **Ngày**: 2026-10-08 · **Trạng thái**: DỰ THẢO chờ duyệt CH1 (kèm w1-06)
**Nguồn ràng buộc**: `plans/vanguard/REPORT.md` §3.2 (bản chuẩn "REPORT.md v2" do orchestrator cung cấp 2026-10-08 — không nằm trong workspace, nguyên văn được trích tại §2 của charter này); 3 report deps: `reports/w1-01.md`, `reports/w1-02.md`, `reports/w1-03.md`.

> Tài liệu này là **hợp đồng giữa các workflow**: W2 (engine/IR/render/CLI), W3 (adapter Java + rules), W4 (corpus/mutation/audit), W5 (real-fire), W6 (release) đọc và code theo charter, không tự ý đổi. Mơ hồ ở đây = chi phí 10 lần ở W2/W3. Task nào thấy charter thiếu thực tế: **ghi issue vào report + contract comment**, orchestrator quyết — không lén sửa.

---

## Mục lục

1. [Sản phẩm & positioning](#1-sản-phẩm--positioning)
2. [Yêu cầu chuẩn R1–R8 (nguyên văn) + bản đồ phủ](#2-yêu-cầu-chuẩn-r1r8-nguyên-văn--bản-đồ-phủ)
3. [Rules taxonomy R1xx–R6xx](#3-rules-taxonomy-r1xxr6xx)
4. [Language matrix](#4-language-matrix)
5. [Kiến trúc](#5-kiến-trúc)
6. [CLI UX spec](#6-cli-ux-spec)
7. [Chất lượng & gates (R7)](#7-chất-lượng--gates-r7)
8. [Versioning & stability policy](#8-versioning--stability-policy)
9. [Phụ lục A — Quyết định mở chờ CH1 chốt](#9-phụ-lục-a--quyết-định-mở-chờ-ch1-chốt)
10. [Phụ lục B — Rủi ro kiến trúc](#10-phụ-lục-b--rủi-ro-kiến-trúc)
11. [Phụ lục C — Log mâu thuẫn deps đã xử lý](#11-phụ-lục-c--log-mâu-thuẫn-deps-đã-xử-lý)
12. [Phụ lục D — Hợp đồng theo workflow](#12-phụ-lục-d--hợp-đồng-theo-workflow)

---

## 1. Sản phẩm & positioning

### 1.1. Định nghĩa một câu

**Vanguard là linter quét trực tiếp SOURCE CODE đa ngôn ngữ, tự phát hiện API surface (REST + gRPC), và kiểm tra thiết kế API theo chuẩn AIP-style (resource-oriented design) — cho ra finding có vị trí `file:line:col`, suggestion sửa cụ thể, output CI-native (pretty / JSON / SARIF 2.1.0).**

- Đầu vào: thư mục source bất kỳ (`vanguard scan /path`) — **không cần build project, không cần spec, không cần chỉ định file** (R1, R4).
- Đầu ra: findings nhóm theo rule/severity, phân loại ERROR/WARN/INFO, exit code 0/1/2 cho CI (R5).
- Đối tượng kiểm tra: **thiết kế API** (naming, verb semantics, pagination, payload schema, error model, versioning, gRPC conventions) — không phải code style (không cạnh tranh Checkstyle/golangci-lint ở tầng style).

### 1.2. Positioning (dẫn từ w1-02)

w1-02 §3 chứng minh khoảng trống: **không tool nào parse Java/Spring source để kiểm tra chuẩn thiết kế API** — spectral/speccy chỉ lint spec đã sinh (muộn một vòng, location không map về Java), protolint/buf chỉ nhận proto, oasdiff chỉ diff 2 spec, golangci-lint khóa Go toolchain. Từ đó:

> **Source-first · design-standard-driven · CI-native.**

| Lớp | Ai giữ | Quan hệ với vanguard |
|---|---|---|
| Governance của **artifact spec** (OpenAPI YAML) | spectral (+ style-guide cộng đồng) | **Bổ trợ** — vanguard không thay thế; springdoc overlay của vanguard (w3-02) chỉ **merge metadata** vào IR, không lint spec |
| Phát hiện **breaking change** giữa 2 version | oasdiff / buf breaking | **Bổ trợ** — differential scan là backlog (§3.7 R6xx-05), không thuộc v0.1 |
| Style/lint **ngôn ngữ** | golangci-lint, Checkstyle, PMD | **Song song, không giao nhau** |
| Governance của **thiết kế API ngay trong source** | **vanguard — độc quyền ở lớp này** | — |

Kế thừa UX: vanguard học **mô hình aggregator** của golangci-lint (config discovery 3 cấp, dual-format, `--new-from-*` backlog) và **exit-code contract 3 mức** mà cả 6 tool khảo sát đều hội tụ (w1-02 §2.1, Phụ lục A). Kế thừa engine: **mô hình rule của api-linter** (selector + predicate thuần + registry + 3 tầng suppression + panic isolation — w1-01 §5.1), chuyển giao gần 1:1 vì `Lint` là hàm thuần trên node; phần **front-end (parser, IR, output, config discovery) thiết kế mới** vì input là source đa ngôn ngữ chứ không phải proto (w1-01 §5.2).

### 1.3. Vì sao tên **vanguard** + chính sách đặt tên (R8)

- **Ý nghĩa**: *vanguard* = đơn vị tiên phong đi trước lực lượng chính. Tool này đứng **trước vòng spec**: khi OpenAPI chưa tồn tại (chỉ sinh sau build/run — w1-02 §3), vanguard đã quét source và chặn sai thiết kế tại PR. Nói cách khác: vanguard là điểm kiểm **đầu tiên** của quy trình, không phải lớp phủ cuối.
- **Thực dụng**: 1 từ, lowercase, gõ dễ trong CLI (`vanguard scan .`), không có ký tự đặc biệt, đọc được trong cả pipeline CI lẫn tài liệu.
- **Chính sách naming (ràng buộc từ R8)**:
  - Brand: **vanguard**. Repo: `github.com/Stellarhold170NT/vanguard`.
  - **Cấm** hardcode tên tổ chức trong rule ID, trong code, trong message. Rule ID là `R<F>xx-NN` trung lập (§3.0). Framework/org-specific detection (vd annotation bảo vệ riêng của một tổ chức) chỉ được vào qua **config** (danh sách annotation cấu hình được trong `.vanguard.yaml`), không vào rule ID hay logic cứng.
  - Đồ họa/docs dùng `vanguard` — không dùng biến thể `VTS-*` (đích danh cái tên bị cấm bởi R8).
  - Go module path: đề xuất `github.com/Stellarhold170NT/vanguard` (quy ước Go cho repo công khai — w2-01 cho phép chọn; chốt ở Phụ lục A-2).

### 1.4. Nằm ngoài phạm vi v0.1 (nói rõ để không ai chờ nhầm)

- **Autofix sửa file** — v0.1 chỉ có `suggestion` (text thay thế máy đọc được, dùng cho quick-fix IDE qua SARIF). Không `--fix`. (nguyên tắc protolint/w1-02 #9: không tự sửa cái có thể đổi hành vi API).
- **Differential mode** (`--new-from-*`) — backlog (w1-02 #10), không block v0.1.
- **Type-resolution đầy đủ** (Java symbol solver) — heuristic + severity kỷ luật thay thế (§5.5).
- **LLM trong đường lint** — loại ở bước so sánh kiến trúc (§5.5, phương án C).
- **UI/web dashboard** — ngoài chương trình; chỉ SARIF/JSON contract cho máy đọc.

---

## 2. Yêu cầu chuẩn R1–R8 (nguyên văn) + bản đồ phủ

**Nguồn**: REPORT.md v2 (§3.2) — bản chuẩn do orchestrator cung cấp 2026-10-08; nguyên văn, không diễn giải lại:

```
R1 | Quet SOURCE CODE (khong bat buoc proto/openapi), TU PHAT HIEN API surface, khong can chi dinh file | vanguard scan tren repo la: phat hien >= 95% endpoint ma swagger liet ke, do tren military-youth
R2 | Kiem tra nguyen tac thiet ke API: REST/HTTP + gRPC, bat nguon tu AIP (aip.dev) nhung viet lai cho source code | >= 25 rule thuoc 6 ho R1xx-R6xx, moi rule co doc + vi du dung/sai
R3 | Framework theo ngon ngu: Java -> Spring Boot (MVC + WebFlux + springdoc); Go -> net/http, gin, chi, grpc-go; Python -> FastAPI, Django REST, Flask | Kien truc adapter cho ca 3; trien khai Java o v0.1; Go/Python wave-2
R4 | CLI chay tu bat ky dau, khong phu thuoc project build | binary tinh (goreleaser, linux/darwin/windows), vanguard scan /path khoi dong <= 3s
R5 | Dau ra chuyen nghiep: pretty (mau, nhom theo rule/severity, hint sua), JSON, SARIF 2.1.0 | SARIF pass schema; pretty la bo mat san pham; explain per-rule; exit codes 0/1/2
R6 | Cau hinh duoc: bat/tat rule, include/exclude path, severity, suppression | .vanguard.yaml + suppression inline vanguard:ignore
R7 | Chat luong den tu thuc chien: adversarial + mutation corpus quy mo lon, chay that tren military-youth | >= 150 fixture case, mutation catch-rate >= 90%, FP/FN audit co so lieu
R8 | KHONG dat ten vts-api-linter — san pham la cong nghe loi dung chung | Brand vanguard, repo GitHub Stellarhold170NT/vanguard, khong hardcode ten to chuc trong rule ID
```

**Bản đồ phủ — mỗi R được trả lời ở mục nào của charter:**

| R | Trả lời tại | Tóm tắt cam kết |
|---|---|---|
| R1 | §5.2 (IR), §5.3 (discovery + adapter), §6 (CLI) | `vanguard scan /path` tự detect ngôn ngữ + framework, tự dựng ApiSurface; cam kết định lượng ≥95% recall so swagger → đo ở W5 (w5-03), thiết kế phục vụ từ IR (mọi node giữ Location) |
| R2 | §3 (taxonomy) | 6 họ R1xx–R6xx; **26 rule in-scope v0.1**, 12 backlog — tổng 38 ứng viên có id + severity + nguồn AIP; mỗi rule có doc + ví dụ đúng/sai (w3-07 sinh từ metadata) |
| R3 | §4 (language matrix) | Adapter architecture cho cả 3; **Java Spring Boot (MVC + WebFlux annotation + springdoc overlay) triển khai ở W3 = v0.1**; Go (net/http, gin, chi, grpc-go) và Python (FastAPI, DRF, Flask) = wave-2 backlog (đăng ký ở w7-04) |
| R4 | §5.5 (chọn phương án), §5.1 (layout), §6.3 | Static Go binary, goreleaser linux/darwin/windows (w6-01), grammar tree-sitter nhúng compile-time; khởi động ≤3s đo trong sandbox (w2-06 acceptance); không đụng Maven/Gradle |
| R5 | §6 (CLI UX spec) | 3 format + mock chính xác để w2-05/w2-06 code theo; SARIF 2.1.0 schema-valid (validate trong CI); `explain` per-rule; exit 0/1/2 |
| R6 | §3.0 (suppression), §6.4–6.5 (config) | `.vanguard.yaml` schema đầy đủ; inline `vanguard:ignore`; 3 tầng precedence như api-linter nhưng + auto-discovery |
| R7 | §7 (chất lượng) | ≥150 adversarial case (w4-01), mutation catch-rate ≥90% (w4-02), FP/FN audit có số liệu (w4-04), real-fire trên military-youth (W5); phương pháp chi tiết = w1-05 |
| R8 | §1.3 (naming policy) | Brand vanguard; rule ID trung lập `R<F>xx-NN`; org-specific chỉ qua config |

---

## 3. Rules taxonomy R1xx–R6xx

### 3.0. Quy ước chung (bắt buộc đọc cho W3)

- **ID format**: `R<F>xx-<NN>` — `<F>` = chữ số họ (1..6), `xx` là **literal** (đọc là "one-ex-ex", chỉ band 1xx..6xx), `<NN>` = số thứ tự 2 chữ số trong họ. Ví dụ thật: `R1xx-02`. ID không chứa tên org (R8). Slug kebab-case đi kèm: `R1xx-02 no-verb-path` — `explain`/config nhận cả ID lẫn prefix họ (`R1xx`, `R2xx`).
- **Severity mặc định**: `ERROR` (thiết kế sai chắc chắn, gây hỏng contract), `WARN` (vi phạm convention có bằng chứng), `INFO` (khuyến nghị/convention mềm). Chỉ ERROR ảnh hưởng exit code (§6.3).
- **Message**: **tiếng Anh**, 1 câu, lý do trước — suggestion là text thay thế cụ thể (mô hình Problem của api-linter: message 1 câu + suggestion máy đọc + location chính xác, w1-01 §2.2). Message tiếng Việt không chuẩn (open decision A-1, khuyến nghị Anh).
- **Suggestion**: nội dung **đúng** để thay vào span (vd `PUT /api/.../movement-reports/{id}`), không phải mô tả cách sửa. Có suggestion → phải có location chính xác (node con, không trỏ cả method).
- **Suppression**: mỗi finding hiển thị dòng `suppress: // vanguard:ignore <rule-id> <reason>` (pretty, 1 lần mỗi nhóm rule). Cơ chế đầy đủ ở §6.5.
- **Metadata là data**: mỗi rule = 1 bản ghi metadata (id, slug, category, severity default, aip ref, summary, ví dụ good/bad, doc path, flags: `defaultDisabled`, `fixable:false`) sống trong data file YAML dưới `rules/`, nhúng vào binary lúc build (go:embed) — binary tĩnh không tải gì lúc runtime (R4). Check function là Go, registry ghép metadata↔check theo id. Engine chỉ import registry; **không** import từng rule (w2-04).
- **Nguồn AIP**: cột "AIP" ghi AIP gốc truyền cảm hứng; rule được **phát biểu lại bằng khái niệm Spring/source** (w1-01 §5.2.4 — không dịch từng rule proto). AIP-193/AIP-180 **không có** rule tương ứng trong api-linter (w1-01 §3.1) → R5xx/R6xx là thiết kế mới, không "copy list".

### 3.1. Họ R1xx — Resource & naming

Nguồn tinh thần: AIP-121 (resource types), AIP-122 (names), AIP-127 (HTTP semantics của resource path), AIP-131/136 (standard/custom methods), AIP-140 (naming). Bối cảnh Spring: w1-03 §2.1 (23/24 controller theo `/{plural}`), §3 pain 1–3.

| ID | Slug | Sev | AIP | Mô tả (1 dòng) | Tình huống Spring thực tế (w1-03) |
|---|---|---|---|---|---|
| R1xx-01 | `plural-collection` | WARN | 131, 122 | Collection path phải dùng danh từ số nhiều (`/training-plan` → `/training-plans`). | 23/24 controller JHipster đã chuẩn `/api/military-youth/<số-nhiều>` (w1-03 §2.1); controller mới lẻ loi đặt singular là lệch ngay so với chuẩn sẵn có — rule chặn trước khi FE sinh 2 bộ đường. |
| R1xx-02 | `no-verb-path` | ERROR | 131, 133, 135 | Path không chứa động từ CRUD (`/delete`, `/update/draft`, `/getUserById`) — HTTP verb đảm nhiệm hành động. | w1-03 pain 1+3: 7× `/search/all-by-condition`, 5× `@DeleteMapping("/delete")` (DefenseDiplomacyResource.java:215, TrainingPlanResource.java:219…), `@PostMapping("/update/draft")` + `/update/submit` (MovementReportResource.java:134,145). Cùng một thao tác xoá tồn tại 2 shape endpoint trong 1 app. |
| R1xx-03 | `resource-path-pattern` | WARN | 127, 131 | Path tài nguyên theo `/collection/{id}/sub-collection/{subId}`; route phẳng kiểu `/find-by-id/{id}` thành `/collection/{id}`. | w1-03 pain 1: `/find-by-id/{id}`, `/find-all-id-active` phá hierarchy tài nguyên; chuẩn JHipster 42 mapping `/{id}` là nền đối chiếu (pain 3). |
| R1xx-04 | `path-casing` | WARN | 122 (camel-case-uris phóng tác) | Segment path kebab-case, biến path camelCase — nhất quán trong toàn app. | w1-03 pain 2: `@GetMapping("/search/all-by-Name")` — chữ N hoa giữa kebab (MagazineArticlesResource.java:55), đối chiếu 7 nơi `/search/all-by-condition` đúng. |
| R1xx-05 | `id-field-naming` | INFO | 122, 140 | Id/resource-name field nhất quán: `id` cho self, `<resource>Id` cho tham chiếu — phát hiện dùng trộn, không ép chuẩn trong v0.1. | w1-03 §2.4: 182 DTO trộn 3 quy ước đặt tên; path variable `/{id}` chuẩn (42 mapping) vs field `requestId`/`youId`-style trong payload — chưa có quy ước id chung. |

### 3.2. Họ R2xx — Methods & HTTP semantics

Nguồn: AIP-131/132/133/134/135 (standard methods), AIP-136 (custom methods). Bối cảnh: w1-03 pain 3, 7.

| ID | Slug | Sev | AIP | Mô tả (1 dòng) | Tình huống Spring thực tế (w1-03) |
|---|---|---|---|---|---|
| R2xx-01 | `get-no-body` | ERROR | 131 | `@GetMapping` không được khai báo tham số `@RequestBody`. | w1-03 pain 7a: `@GetMapping("/auto-complete") autoComplete(@RequestBody YouthAutocompleteRequest …)` (YouthResource.java:113-117) — HTTP client phổ biến bỏ body của GET → endpoint "mất" input lặng lẽ. |
| R2xx-02 | `post-creates-201` | WARN | 133 | POST tạo resource mới phải hướng tới 201 (return `ResponseEntity` + status 201 hoặc `@ResponseStatus(CREATED)`). | w1-03 pain 3 (đối chiếu): chuẩn JHipster của repo trả 201 cho create; endpoint POST mới trả 200 lẻ loi là lệch contract — rule bảo vệ convention đã có trong app. |
| R2xx-03 | `patch-partial` | WARN | 134 | PATCH nhận payload partial (heuristic: body type ≠ entity/DTO full của response); full update nên là PUT. | w1-03 pain 7c: app trộn PUT-partial (`partialUpdateYouth`, YouthResource.java:62-64) với 4 PATCH thật nơi khác — 2 cách update một concept sống chung. |
| R2xx-04 | `delete-no-body` | WARN | 135 | `@DeleteMapping` không khai báo `@RequestBody` (xoá nhiều → dùng batch path hoặc query params). | w1-03 pain 7b: `@DeleteMapping` + `@RequestBody List<String> ids` (YouthResource.java:76-79) — 5 chỗ `/delete` nữa cùng pattern (pain 3). |
| R2xx-05 | `custom-method-post` | INFO | 136 | Endpoint hành động (không CRUD) nên là POST + path `:verb` (AIP-136 style: `POST /books/{id}:archive`). | w1-03 pain 1: `/generate-code`, `/get-term` là hành động; đề xuất shape `POST /{id}:generate-code` thay vì verb-in-path — khớp 10× `/export`, 4× `/download-template-import`. |
| R2xx-06 | `put-full-update` | WARN | 134 | PUT nhận full entity (partial update → PATCH); heuristic: body cùng type với response/DTO chính. | w1-03 pain 7c: `partialUpdateYouth` dùng PUT (YouthResource.java:62-64) — client không phân biệt được PUT/PATCH ở app này. |

### 3.3. Họ R3xx — Pagination & collections

Nguồn: AIP-158 (pagination), AIP-132 (List), AIP-160 (filter — backlog). Bối cảnh: w1-03 pain 4, 9.

| ID | Slug | Sev | AIP | Mô tả (1 dòng) | Tình huống Spring thực tế (w1-03) |
|---|---|---|---|---|---|
| R3xx-01 | `list-paginated` | WARN | 158 | List endpoint (GET collection + trả list) phải nhận tham số phân trang (`Pageable` / `page,size` / config pattern). | w1-03 pain 4 nền tảng: 46 file import `Page`, 13 endpoint trả `Page<>` — list không phân trang xen giữa là nguy cơ dump table (endpoint `/export` 10× càng trầm trọng hơn). |
| R3xx-02 | `list-envelope` | INFO | 158 | Response list nên bọc envelope có items + metadata (tổng số, page) thay vì mảng trần. | w1-03 pain 4: hai hình `Page<T>` (Spring) vs `PageResponse` (wrapper jar riêng tư) — cả hai đều có envelope; mảng trần không có chỗ cho metadata. |
| R3xx-03 | `unbounded-page-size` | INFO | 158 | Tham số size không có cap (`@Max`/validation) → cảnh báo DoS nhẹ. | w1-03 pain 4: Pageable dùng rộng (46 file) nhưng không thấy evidence cap — rule kiểm tra `@Max` trên `size`/`limit` do dev tự khai. |
| R3xx-04 | `pagination-consistency` | WARN | 158 | Trong 1 codebase, các list endpoint dùng nhiều hình phân trang không tương thích → cảnh báo + liệt kê từng hình. | w1-03 pain 4 trực diện: 13 endpoint trả `ResponseEntity<Page<...>>` (YouthResource.java:93-95) vs 19 chỗ `PageResponse` jar riêng tư (ReportResource, UnionMembershipExpenditureResource…) — 2 JSON pagination khác nhau sống chung 1 API. |

### 3.4. Họ R4xx — Payload & DTO schema

Nguồn: AIP-140 (field naming), AIP-141 (forbidden types), AIP-142 (time fields), AIP-143 (standardized codes), AIP-203 (field behavior). Bối cảnh: w1-03 pain 6, §2.4.

| ID | Slug | Sev | AIP | Mô tả (1 dòng) | Tình huống Spring thực tế (w1-03) |
|---|---|---|---|---|---|
| R4xx-01 | `no-entity-in-payload` | ERROR | 121, 203 | Payload/response không dùng type entity ORM (heuristic: `@Entity`/jakarta.persistence annotation trong scope scan, hoặc package pattern config). | w1-03 pain 6: `@GetMapping("/current")` trả thẳng entity `Youth` (YouthResource.java:121-124); import trả `ImportResult<Youth>` (YouthResource.java:83-87) — expose schema DB + lazy-loading ra API. |
| R4xx-02 | `field-casing` | WARN | 140 (lower-snake → JSON camelCase) | DTO field trong payload JSON phải camelCase nhất quán (không trộn snake_case). | w1-03 §2.4: 182 DTO file với conventions trộn lẫn — field casing lộn xộn là hệ quả tự nhiên cần chặn ở DTO mới. |
| R4xx-03 | `time-field-standard` | INFO | 142 | Field thời gian dùng `Instant`/`OffsetDateTime` (serialization RFC3339), không `String`/kiểu phi chuẩn. | Không có hit trực tiếp trong recon (ghi minh bạch); rủi ro có thật vì 182 DTO tự viết — rule đề phòng, tôn trọng convention ghi trong `.vanguard.yaml`. |
| R4xx-04 | `dto-naming-convention` | INFO | 140 | DTO dùng 1 quy ước hậu tố (`*DTO` vs `*Request`/`*Response` vs record) — phát hiện dùng trộn theo pattern config. | w1-03 §2.4: 182 file DTO chia `projection/`, `response/`, `record/`, `request/` + hậu tố `DTO` — 3 quy ước sống chung khiến FE không đoán được tên type. |

### 3.5. Họ R5xx — Errors & status semantics

Nguồn: AIP-193 (error model — **thiết kế mới**, api-linter không có — w1-01 §3.1), AIP-191 (clarity). Bối cảnh: w1-03 pain 5.

| ID | Slug | Sev | AIP | Mô tả (1 dòng) | Tình huống Spring thực tế (w1-03) |
|---|---|---|---|---|---|
| R5xx-01 | `unified-error-shape` | WARN | 193 | `@ExceptionHandler` trả cấu trúc lệch ErrorScheme chuẩn của app (config: `problem+json` hoặc `{code,message,details}`). | w1-03 pain 5: app có `ExceptionTranslator` theo RFC-7807 (web/rest/errors/ExceptionTranslator.java:42,45) — handler mới trả `Map`/wrapper riêng là vỡ contract lỗi chung. |
| R5xx-02 | `no-500-for-business` | ERROR | 193 | Exception nghiệp vụ (tên/inheritance gợi ý business) không được map 500. | Heuristic dựa trên cấu trúc `@ExceptionHandler` (w3-02 đưa ErrorScheme vào IR); FP-an toàn: chỉ ERROR khi tín hiệu tên rõ (vd hậu tố `Exception` + package `service`). |
| R5xx-03 | `status-semantics` | WARN | 133, 193 | POST tạo resource trả 200 thay vì 201/202 (phủ từ phía status; cặp với R2xx-02). | w1-03 pain 3 (nền): app dựa convention JHipster 201 cho create; endpoint mới lệch là regress không tiếng. |
| R5xx-04 | `single-error-advice` | WARN | 193 | Nhiều hơn một nguồn error envelope toàn cục (`@ControllerAdvice`/`@ExceptionHandler` chồng) hoặc 2 envelope khác nhau trong 1 app. | w1-03 pain 5: `ExceptionTranslator` (RFC-7807) + `@ControllerAdvice` thứ hai `ResponseWrapper` bọc mọi 2xx (config/ResponseWrapper.java:39) — client parse 2 contract tuỳ 2xx/4xx; bonus bug thật trong `supports()` (ResponseWrapper.java:21-22). |

### 3.6. Họ R6xx — Versioning & multi-protocol

Nguồn: AIP-180 (backwards compatibility — **thiết kế mới**, w1-01 §3.1), AIP-215 (versioning), AIP-127; gRPC qua so khớp Standard Methods. Bối cảnh: w1-03 §2.1 (không `/v1`), §2.3 (11 service gRPC / 88 rpc).

| ID | Slug | Sev | AIP | Mô tả (1 dòng) | Tình huống Spring thực tế (w1-03) |
|---|---|---|---|---|---|
| R6xx-01 | `versioned-path` | WARN · **mặc định TẮT** | 215, 180 | Base path chứa segment version (`/v1`) theo pattern config (mặc định `/v[0-9]+`). | w1-03 §2.1: military-youth **không có `/v1` nào** — bật mặc định sẽ flood WARN trên cả app; đây là **policy rule** (chọn chiến lược versioning), không phải defect → opt-in qua config. |
| R6xx-02 | `grpc-standard-methods` | INFO | 131–136 (áp cho rpc) | rpc trong `.proto` nên theo Standard Methods naming (`Get/List/Create/Update/Delete` + `VerbNoun`); đọc khai báo service/rpc tối giản. | w1-03 §2.3: 11 service / 88 rpc khai báo trong `src/main/proto` (2 service bị comment out phải bỏ qua — commented code không phải API surface); tên rpc lệch chuẩn ngay từ proto sẽ lan xuống 10 class impl `web/grpc/service/*GrpcService.java`. |
| R6xx-03 | `version-consistency` | INFO | 215, 180 | Nếu version segment tồn tại ở một số endpoint thì các endpoint cùng service prefix phải cùng version — phát hiện migration nửa vời. | Mở rộng trực tiếp từ w1-03 §2.1 (base path chuẩn ×23 nhưng RoutesController dùng `/api` — một base path lẻ loi là tiền đề của version/route lệch). |

### 3.7. Backlog (ngoài v0.1 — có id trước, vào theo ratchet w7-04)

| ID (dự kiến) | Slug | Sev | AIP | Vì sao để backlog |
|---|---|---|---|---|
| R1xx-06 | `search-convention` | INFO | 132 | w1-03 pain 9 cho thấy 4 kiểu search trong 1 app — nhưng ép 1 convention trên legacy = FP ồ ạt; cần differential mode trước. |
| R1xx-07 | `path-abbreviations` | INFO | 140 | Danh sách viết tắt chuẩn (config-driven); ít meat hơn các rule trên. |
| R2xx-07 | `method-signature` | INFO | 132 | AIP-132 `method_signature` — cần inference tham số tốt hơn v0.1. |
| R3xx-05 | `filter-field-name` | INFO | 160 | Filter field chuẩn `filter`; cần convention config trước. |
| R3xx-06 | `filter-type-standard` | INFO | 160 | Kiểu AIP-160 filter syntax — cần mini-language; phức tạp hơn giá trị v0.1. |
| R4xx-05 | `standardized-codes` | INFO | 143 | `country`→`region_code`, `content_type`→`mime_type` (chuyển ngữ từ w1-01 §2.2, rule 143) — danh sách chuẩn hoá dài, phù hợp sau khi corpus ổn. |
| R4xx-06 | `forbidden-field-types` | INFO | 141 | Không dùng kiểu "túi đựng mọi thứ" (Object/Map<String,Object>) trong payload — heuristic tiếng ồn cao. |
| R5xx-05 | `error-details-shape` | INFO | 193 | `details` của error phải có type/shape ổn định — phụ thuộc ErrorScheme maturity. |
| R5xx-06 | `retryable-error-info` | INFO | 193 | Thông tin retry (AIP-193 retryable) — chỉ có giá trị khi error model đã thống nhất. |
| R6xx-04 | `mutation-authorized` | WARN | — (security-adjacent) | **Ghi nhận từ w1-03 pain 10**: `@VauthzCheck(action = XEM_CT)` trên `@DeleteMapping` (YouthResource.java:76-79, annotation dòng 78) + `@VauthzCheck` bị comment out trên `GET /filter` (YouthResource.java:91). Rule: mọi endpoint mutating phải có annotation authz từ **danh sách cấu hình được** (không hardcode org — R8). Để backlog vì annotation authz là org-specific; vào khi config schema có `authzAnnotations` + W5 FP data. |
| R6xx-05 | `breaking-change-diff` | WARN | 180 | Differential scan giữa 2 revision (w1-01 §3.1: ngoài phạm vi api-linter; w1-02 #10: dạng oasdiff). Cần baseline infra — sau v0.1. |
| R6xx-06 | `grpc-field-casing` | INFO | 140 | Field naming trong `.proto` — cần proto parser thật hơn regex của w3-06. |

**Tổng ứng viên: 38 (26 in-scope + 12 backlog) — thỏa R2 (≥25 rule thuộc 6 họ, mỗi rule có doc + ví dụ đúng/sai — w3-07 sinh từ metadata).**

### 3.8. Thêm rule mới sau v0.1 (governance pointer)

Mọi rule mới đi qua quy trình CONTRIBUTING (w1-06) + phải có: metadata đầy đủ, ≥1 fixture vio + ≥1 ok (quy ước `testdata/adversarial/<rule-id>/` của w4-01), doc `docs/rules/<id>-<slug>.md`, cross-check `--list-rules` vs docs (w3-07). Thêm/bớt rule = minor bump (§8). Rule "lén" thêm ngoài charter không được — ghi đề xuất vào report như w3-03 protocol đã nói.

---

## 4. Language matrix

| Ưu tiên | Ngôn ngữ | Framework adapter | Phát hành | Lý do thứ tự |
|---|---|---|---|---|
| **1** | **Java** | **Spring Boot MVC** (annotation model: `@RestController`, `@…Mapping`, `@PathVariable/RequestParam/RequestBody`, `@ResponseStatus`, `@ControllerAdvice`) · **WebFlux** (cùng annotation model; return type `Mono<X>`/`Flux<X>` được unwrap) · **springdoc overlay** (merge operationId/schema names, không bắt buộc) | **v0.1 (W3)** | (a) Fixture thật duy nhất của chương trình là Spring Boot 4.0.2 + JHipster 8.8 (w1-03 §4.1) — R1 (recall ≥95% vs swagger) và R7 (corpus/real-fire) **đo được trên chính nó**; (b) 10 điểm đau w1-03 §3 đã mapped 1-1 vào taxonomy §3 — rule có "meat" từ ngày đầu; (c) AIP resource-oriented mapping tự nhiên nhất với Spring annotation model; (d) springdoc 2.7.0 có sẵn trong fixture cho overlay (w1-03 §2.2). |
| 2 | Go | net/http (ServeMux patterns), gin, chi; grpc-go | **wave-2** (backlog — đăng ký ratchet ở w7-04) | Handler function signature tường minh + route tree tường minh (gin/chi) → adapter ít variance; grpc-go là cầu nối tự nhiên cho họ R6xx; tree-sitter-go trưởng thành. Nhưng **không có fixture thật** trong chương trình → cần mini-recon (kiểu w1-03) trước khi viết adapter. |
| 3 | Python | FastAPI, Django REST Framework, Flask | **wave-2** (backlog) | Ba idioms khác hẳn nhau (decorator + type hints / class-based ViewSet / blueprint route) → variance adapter cao nhất; cần recon riêng; đặt cuối để không làm chậm wave-1. |

**Điều kiện wave-2 (ràng buộc cho w7-04 backlog)**: mỗi ngôn ngữ phải có (1) mini-recon report trước, (2) fixture repo thật hoặc tổng hợp đạt chuẩn w4-01, (3) adapter pass golden harness w2-07 **không sửa harness** — chứng minh kiến trúc adapter thực sự gọn. Taxonomy R1xx–R6xx **không đổi** khi thêm ngôn ngữ; chỉ adapter + framework metadata mới.

**Scope ghi rõ cho v0.1 (Java)**: MVC annotation model + WebFlux annotation (Mono/Flux unwrap) + springdoc overlay tùy chọn; **không** lint functional `RouterFunction` (backlog — Phụ lục A-7); gRPC chỉ đọc khai báo `.proto` mức service/rpc (w3-06), full gRPC rules → wave-2.

---

## 5. Kiến trúc

### 5.1. Sơ đồ + repo layout

```
                        ┌─────────────────────────────────────────┐
                        │               cmd/vanguard              │
                        │      cobra CLI · subcommands (§6.1)     │
                        │      R4: binary tĩnh, start ≤ 3s        │
                        └──────┬──────────────────────────┬───────┘
                               │                          │
                      ┌────────▼────────┐        ┌────────▼────────┐
                      │    discovery    │        │     render      │
                      │ walker + detect │        │ pretty/json/    │
                      │ adapter registry│        │ sarif (§6.6-8)  │
                      └────────┬────────┘        └────────▲────────┘
                               │ files + detect evidence   │ findings (sorted, deterministic)
                      ┌────────▼────────┐        ┌────────┴────────┐
                      │   adapters/*    │  IR    │     engine      │
                      │ java (W3)       ├───────►│ selector +      │
                      │ go/py (wave-2)  │ApiSurface│ rule registry │
                      └─────────────────┘ (§5.2) │ suppression     │
                                                 └────────┬────────┘
                                                          │ đọc
                                                 ┌────────▼────────┐
                                                 │  rules/ (data)  │
                                                 │ metadata YAML   │
                                                 │ + check Go      │
                                                 │ R1xx…R6xx (§3)  │
                                                 └─────────────────┘
```

Repo layout (khớp w1-06 deliverable 2 + w2-01; không ai tự chế thêm nhánh package):

```
vanguard/
├── cmd/vanguard/main.go          # entry — gọi internal/cli
├── internal/
│   ├── ir/                       # ApiSurface IR (w2-02) — thuần, không phụ thuộc tree-sitter
│   ├── engine/                   # rule engine + config + suppression (w2-04)
│   ├── render/                   # pretty / json / sarif (w2-05)
│   ├── discovery/                # walker + language detect + adapter registry (w2-03)
│   ├── cli/                      # cobra wiring (w2-06)
│   └── golden/                   # golden snapshot harness (w2-07)
├── adapters/java/                # tree-sitter-java bóc thô (w3-01) + spring mapping (w3-02)
├── rules/                        # R1xx–R6xx: metadata data file + check Go + registry (w3-03..06)
├── testdata/                     # stub-repo (w2-03) · golden (w2-07) · adversarial (w4-01) · mutation (w4-02)
├── docs/                         # charter.md · test-strategy.md · rules/<id>-<slug>.md (w3-07)
├── reference/api-linter/         # clone googleapis/api-linter để đọc — .gitignore (w1-06)
└── .github/workflows/ci.yml      # build + test + vet + SARIF schema validate (w1-06)
```

### 5.2. IR `ApiSurface`

Thiết kế chi tiết thuộc w2-02; charter chốt **node inventory + bất biến + ma trận phụ thuộc rule**. Bất biến nền: **mọi node giữ `Location{File,Line,Column}` đầy đủ** (finding phải trỏ được file:line — UC1/R1); IR thuần, **không** import tree-sitter hay ngôn ngữ cụ thể (w2-02 acceptance); marshal/unmarshal JSON ổn định để golden snapshot (w4-03 determinism).

Node inventory (tên type theo w2-02):

```
ApiSurface
├── Source{Lang, Framework, FrameworkVersion}      # vd java / spring-boot / 4.0.2
├── Services[]            # 1 Service = @RestController class (REST) hoặc service proto (gRPC)
│   ├── Name · BasePath (merged, normalized) · Annotations{raw} · Location
│   └── Methods[]
│       ├── OperationName (tên method Java/rpc) · Verb (GET|POST|PUT|PATCH|DELETE|rpc)
│       ├── Path (full sau merge base+method — w3-02 test double-slash)
│       ├── Params[] (name, in: path|query|header, TypeRef, validation)
│       ├── Payload (request body TypeRef, nullable)
│       ├── Response{TypeRef, StatusCode, IsCollection, Envelope?}
│       ├── Pagination{Style: pageable|params|none, PageSizeCapped bool}
│       ├── Annotations{raw — @ResponseStatus, authz, ...}
│       └── Location{File,Line,Column}
├── Types[]               # DTO/POJO/record trong scope scan
│   ├── Name · Kind (record|pojo) · Package · IsEntity (persistence annotation) · Location
│   └── Fields[] (Name, JsonName, TypeRef, Annotations{raw}, Location)
├── ErrorScheme
│   └── Handlers[] (ExceptionType, ResponseType, StatusCode, Location)   # từ @ControllerAdvice (w3-02)
├── GrpcServices[]        # đọc mức khai báo .proto: Name, Rpcs[]{Name}, Location (w3-06)
└── Diagnostics[]         # parse errors, annotation unresolved — best-effort, KHÔNG fatal
```

Ma trận phụ thuộc rule → IR (input cho w2-02 trước khi code):

| Họ rule | IR fields cần |
|---|---|
| R1xx | Method.Path (merged), Service.BasePath, Method.Verb, Types[].Fields (Name/JsonName), Location |
| R2xx | Method.Verb, Params (in=body), Response.StatusCode (+@ResponseStatus qua Annotations), Payload.TypeRef, Response.TypeRef |
| R3xx | Method.Pagination, Params (page/size), Response.IsCollection + Envelope |
| R4xx | Types[].Fields (JsonName, TypeRef, Annotations), Types[].IsEntity + Package, Payload/Response.TypeRef |
| R5xx | ErrorScheme.Handlers, Response.StatusCode, Annotations (@ResponseStatus) |
| R6xx | Service.BasePath (segment version), GrpcServices[] (Name, Rpcs), Location |

Comment extraction: adapter bóc comment (line/block) kèm Location để engine match suppression (§6.5) — không đọc lại file trong rule (w3-05 acceptance).

### 5.3. Discovery + Adapter interface

- **Walker** (w2-03): bỏ `.git`, `vendor/`, `node_modules/`, `target/`, `build/`, thư mục ẩn; tôn trọng `.vanguardignore`; giới hạn depth/size (hằng số công bố); không theo symlink vòng; mọi path relative từ scan root, chặn `..`.
- **Language detect** theo extension + build file (`pom.xml`/`build.gradle` → java; `go.mod` → go; `requirements.txt`/`pyproject.toml` → python) → ranked candidates + **evidence** (in ở verbose).
- **Adapter interface** (w2-03):

```go
type Adapter interface {
    Language() string
    Detect(files []string) (bool, evidence)
    Parse(files []string) (*ir.ApiSurface, diagnostics)   // best-effort, KHÔNG abort
}
```

- Adapter đăng ký qua registry; scan chọn adapter theo detect; chọn sai/sparse → evidence trong verbose để debug.
- **Best-effort contract (bắt buộc, khác api-linter)**: api-linter compile-gate — proto lỗi parse là abort toàn bộ (w1-01 §1.2). Source sống **luôn** có file không parse được (macro, codegen chưa chạy) → vanguard: 1 file lỗi → diagnostics + skip, không ảnh hưởng file khác, **không đổi exit code**; parse diagnostics hiện ở verbose + summary count. Điểm khác biệt nguyên tắc lớn nhất so với cha tinh thần (w1-01 §5.2.1).

### 5.4. Rule engine (data-driven, theo w1-01)

- **Rule = metadata (data YAML, go:embed) + check (Go func)** ghép bằng registry. Check signature (nguyên văn w2-04): `func(ctx *LintContext, node ir.Node) []Finding`. `Finding`: Location, message, suggestion (optional), ruleId — **engine gán ruleId, rule không tự set** (bài học FIXME trong api-linter, w1-01 §1.2.5).
- **Selector** = filter node IR type + applicability predicate bằng khái niệm framework (vd "Method Verb=GET", "Type.IsEntity", "Service.Framework=spring-boot") — tương đương `OnlyIf` của api-linter (w1-01 §2.1), viết lại bằng khái niệm Spring chứ không dịch `google.api.*` (w1-01 §5.2.4).
- **Helper library dùng chung** bắt buộc trước khi viết rule đầu tiên (vd `isPaginatedList`, `isRestController`, `unwrapResponseEntity`, `mergePath`) — ~40% rule AIP-13x của api-linter chỉ 5–15 dòng nhờ tầng utils; không có tầng này sẽ sinh 20 bản copy của cùng check pagination (w1-01 §5.1.3).
- **Panic isolation per rule**: 1 rule panic → recover thành diagnostic, các rule khác chạy tiếp (w1-01 §5.1.6 — bắt buộc vì tree-sitter query lỗi runtime dễ).
- **Deterministic**: không dựa thứ tự map; findings sort `(file, line, col, ruleId)`; render 2 lần = byte-identical (w2-05, w4-03).
- **Suppression per-problem**: suppression re-check **per finding** chứ không per-rule (1 rule emit nhiều finding ở nhiều node — kinh nghiệm 5 năm của api-linter, w1-01 §5.1.5).
- **Registry validation** (test time): id format §3.0, unique, đúng họ, slug kebab, severity hợp lệ, doc file tồn tại (khớp w3-07), ví dụ good/bad có mặt. Lỗi đăng ký = fail test, không fail runtime.
- **Không autofix trong engine** (§1.4); `fixable` metadata để sau (backlog).

### 5.5. So sánh phương án kiến trúc (≥2 phương án thay thế)

| Tiêu chí (trọng số theo R) | **A. Go core + tree-sitter + IR ApiSurface + rules data-driven (CHỌN)** | B. Linter JVM (JavaParser/Spoon) | C. LLM quét trực tiếp | D. Spec-first (springdoc → spectral) |
|---|---|---|---|---|
| Binary tĩnh, chạy từ bất kỳ đâu, ≤3s (R4) | ✅ goreleaser linux/darwin/windows, grammar nhúng compile-time | ❌ cần JVM + classpath Maven của project | ❌ runtime dịch vụ bên ngoài | ❌ phải build/run app trước để sinh spec |
| Đa ngôn ngữ bằng adapter (R3) | ✅ 1 engine N adapter, IR chung | ❌ engine viết bằng Java → mỗi ngôn ngữ 1 engine riêng (×3 maintenance) | ⚠️ "được" về lý thuyết, không test được | ❌ 1 spec sinh mỗi framework |
| Determinism + đo được (R7) | ✅ pure function trên IR; golden snapshot; mutation catch-rate có nghĩa | ✅ deterministic | ❌ output non-deterministic mỗi lần chạy — golden/mutation/FP/FN audit **vô nghĩa** | ✅ deterministic (nhưng input muộn 1 vòng) |
| Chi phí/latency 10k+ file (w4-03) | ✅ cục bộ, giây-level | ⚠️ JVM boot + model loading | ❌ mỗi PR tốn tiền + chậm | ⚠️ build + spectral |
| Location chính xác file:line (R5) | ✅ tree-sitter node span | ✅ | ❌ LLM trỏ line lệch thường xuyên, không đảm bảo | ❌ trỏ vào YAML sinh ra, không map về Java |
| Bảo mật/riêng tư | ✅ source không rời máy | ✅ | ❌ source rời môi trường (military-youth: code doanh nghiệp, không được rời — w1-03) | ✅ |
| Type-level insight (DTO vs entity, generics) | ⚠️ heuristic + severity kỷ luật + FP audit (W4/W5) | ✅ symbol solver đầy đủ | ⚠️ cảm tính | ⚠️ chỉ thấy gì lọt vào spec |
| Phát hiện rule "intent" trong code (annotation, naming, comment) | ✅ | ✅ | ⚠️ | ❌ generated cruft không phản ánh code intent (w1-02 §3) |

**Verdict**: A là phương án duy nhất thỏa đồng thời R3+R4+R7; nhược điểm type-insight của A được quản bằng heuristic công bố + severity kỷ luật (không chắc → WARN/INFO, w3-04 protocol) + FP audit W4/W5. B bị loại vì vi phạm R4 và nhân bản engine ×3. C bị loại vì phá determinism/R7 + chi phí + riêng tư; LLM chỉ có thể quay lại sau này vai trò **assistive ngoài đường lint** (polish message/enrich explain), tắt mặc định, và không thuộc v0.1. D bị loại làm lõi (w1-02 §3: muộn một vòng, location không map về source) nhưng **giữ giá trị bổ trợ**: W5 cross-check swagger vs IR dùng chính lớp này làm tiêu chuẩn đối chứng (w5-03).

**Quyết định binding tree-sitter** (cgo chính thức vs pure-Go `smacker/go-tree-sitter`): giao w3-01 quyết trên tiêu chí chốt tại đây — (1) build được binary tĩnh không runtime dep, (2) cross-compile 3 OS không hỏng, (3) extract đủ annotation argument raw (string/array/nested), (4) tốc độ đủ perf w4-03. Ghi lựa chọn + lý do vào report w3-01 (w3-01 protocol đã dành chỗ).

---

## 6. CLI UX spec

### 6.1. Subcommands

| Lệnh | Ý nghĩa |
|---|---|
| `vanguard scan [path]` | Quét + in findings (path mặc định `.`). Format mặc định `pretty`. |
| `vanguard check [path]` | Alias scan mode CI: chỉ exit code; **suppress pretty khi non-TTY** (trừ khi `--format json\|sarif` / `--output` được truyền tường minh). |
| `vanguard explain <rule-id>` | In doc + ví dụ đúng/sai của rule từ metadata (nguồn cho w3-07, hiển thị doc URL). |
| `vanguard init` | Sinh `.vanguard.yaml` mẫu có comment đầy đủ (§6.4.3). |
| `vanguard version` | Build info (commit, date — ldflags; fallback `dev` khi không có git info — đừng fail build, w2-06). |

### 6.2. Flags (scan/check)

| Flag | Ý nghĩa |
|---|---|
| `--format pretty\|json\|sarif` | Format output (default `pretty`). |
| `--output FILE`, `-o FILE` | Ghi ra file thay vì stdout. |
| `--config FILE` | Config tường minh — bỏ qua discovery. |
| `--severity ERROR\|WARN\|INFO` | **Ngưỡng hiển thị** (default INFO = hiện tất cả). KHÔNG đổi exit code (display ≠ fail — w1-02 #2). |
| `--no-color` | Tắt màu (mặc định auto: màu chỉ khi TTY; tôn trọng `NO_COLOR` env). |
| `--verbose`, `-v` | Evidence detect, file skipped, parse diagnostics, suppression note, config path. |
| `--rules <id-list>` | Chỉ chạy các rule này (comma-separated: id lẻ hoặc prefix họ `R1xx`). |
| `--list-rules` | In bảng rule (sort + metadata) rồi exit 0 — API công khai cho CI review + w3-07 docs (w1-02 #7). |

`--fail-severity` → backlog (v0.1 fail cố định ở ERROR — §6.3).

### 6.3. Exit codes + stdout/stderr (contract CI)

| Code | Ý nghĩa |
|---|---|
| **0** | Scan hoàn tất, **không finding ERROR** (WARN/INFO được phép; repo không phát hiện API surface cũng là 0 — w2-03 — với thông báo rõ + summary). |
| **1** | Có **≥1 finding ERROR** (sau config/suppression). |
| **2** | Lỗi tool/config: config sai schema (kèm file+dòng), file config không đọc được, lỗi internal. In vào **stderr**, không in findings. |

- **stdout** (hoặc `--output` file): chỉ findings + summary. **stderr**: lỗi tool + verbose diagnostics. Stdout phải thuần machine-readable khi `--format json|sarif` — cấm banner/marketing (w2-06).
- WARN/INFO **không** đổi exit code — hành vi spectral: `check` chỉ nhìn exit code, `scan` in đủ mọi severity (w1-02 Phụ lục A).

### 6.4. Config `.vanguard.yaml`

#### 6.4.1. Schema (v0.1 — `version: 1`)

```yaml
version: 1                          # bắt buộc; v0.1 chỉ nhận 1

frameworks: {}                      # optional; rỗng = auto-detect
  # java: [spring-boot-mvc]         # ép adapter; wave-2: go:[net-http,gin,chi], python:[fastapi,drf,flask]

include: ["**"]                     # glob relative với thư mục CHỨA file config (oasdiff lesson — w1-02 #6)
exclude:                            # áp sau built-in ignore của discovery
  - "**/generated/**"

rules:                              # per-rule override; id lẻ hoặc prefix họ
  R1xx-02:
    severity: WARN                  # ERROR | WARN | INFO
  R4xx-04:
    disabled: true
  R6xx-01:                          # rule mặc định TẮT (§3.6)
    disabled: false
    options:                        # chỉ nhận options khai báo trong metadata rule — sai = exit 2
      versionPattern: "/v[0-9]+"

suppressions:                       # path-scoped (buf `ignore_only` style — w1-02 #8)
  - rule: R1xx-02                   # id hoặc prefix họ
    paths: ["src/legacy/**"]
    reason: "legacy module, refactor W2"
```

#### 6.4.2. Discovery + precedence

1. **Discovery**: `.vanguard.yaml` từ CWD **leo lên tận root** (golangci/protolint chuẩn — w1-02 #5). Tìm thấy file đầu tiên → dùng, **không merge đa cấp** (v0.1 giữ đơn giản).
2. `--config FILE` đè discovery; `--no-config` tắt hẳn; `--verbose` in config path đang dùng (thấy ngay ở header pretty — §6.6).
3. **Precedence trong pipeline** (cao → thấp): CLI flags (`--rules`, `--severity` hiển thị) → config file (disabled → severity → options) → **suppression** áp sau cùng per-finding: inline comment thắng config suppression.
4. Rule enable/disable xử lý cùng cơ chế api-linter: disable match trước, enable sau — match theo id đầy đủ lẫn prefix (`"R1xx"`, `"R1xx-02"` đều hợp lệ — w1-01 §2.3).

#### 6.4.3. Template `vanguard init` (mock rút gọn — template đầy đủ có comment cho từng key và liệt kê đủ 26 rule id)

```yaml
# .vanguard.yaml — generated by `vanguard init` (vanguard 0.1.0)
# Discovery: nearest .vanguard.yaml từ CWD trở lên; CLI flags thắng file này.
version: 1

frameworks: {}                     # rỗng = tự phát hiện (java: spring-boot)
include: ["**"]
exclude: ["**/generated/**"]

rules:
  # R1xx-01..05 · R2xx-01..06 · R3xx-01..04 · R4xx-01..04 · R5xx-01..04 · R6xx-01..03
  R6xx-01:                         # mặc định TẮT (policy rule — versioned path). Bật:
    disabled: false
    severity: WARN
  # R1xx-02:
  #   severity: ERROR              # ví dụ nâng severity

suppressions: []                   # [{rule: R1xx-02, paths: [...], reason: "..."}]
```

### 6.5. Suppression inline

- **Dòng**: `// vanguard:ignore <rule-id> <lý do?>` — comment cùng dòng finding hoặc dòng **liền trên** phát biểu/lệnh khai báo chứa finding.
- **Khối**: `// vanguard:ignore-begin <rule-id>` … `// vanguard:ignore-end <rule-id>`.
- Match id đầy đủ hoặc prefix họ. Adapter bóc comment + Location (§5.2); engine filter per-finding; số suppressions được **đếm** và in ở summary; `--verbose` note từng cái (w2-04). Counter "suppression không còn tác dụng" (w1-02 #8, differentiation rẻ tiền) → backlog w3+ nếu nhẹ, ghi rõ khi bỏ.

### 6.6. Mock output `pretty` (bản vẽ bắt buộc trước khi code — w2-05/w2-06 code theo mock này)

```text
$ vanguard scan ./military-youth

 vanguard 0.1.0 · ./military-youth · java · spring-boot 4.0.2 · config: ./.vanguard.yaml
 24 services · 185 endpoints · 11 grpc services · 26 rules (25 active) · 598 files in 3.2s

 ERROR R2xx-01 get-no-body (3)
 ──────────────────────────────────────────────────────────────────
 src/main/java/com/viettel/tcct/web/rest/YouthResource.java:113:44
   GET endpoint declares @RequestBody — GET must not have a body.
   suggest: use @RequestParam for the fields, or switch to POST /auto-complete
   suppress: // vanguard:ignore R2xx-01 <reason>

 src/main/java/com/viettel/tcct/web/rest/OrderResource.java:58:30
   GET endpoint declares @RequestBody — GET must not have a body.
   suggest: move the payload to @RequestParam

 WARN  R1xx-02 no-verb-path (14)
 ──────────────────────────────────────────────────────────────────
 src/main/java/com/viettel/tcct/web/rest/MovementReportResource.java:134:24
   Path "/update/draft" contains the verb "update" — use PUT/PATCH on the resource path.
   suggest: PUT /api/military-youth/movement-reports/{id}
   suppress: // vanguard:ignore R1xx-02 <reason>

 (…các nhóm rule tiếp theo: WARN R3xx-04 pagination-consistency (5), INFO R4xx-04 …)

 ──────────────────────────────────────────────────────────────────
 26 findings · 3 ERROR · 14 WARN · 9 INFO · 12 suppressed
 5 files with findings · 598 scanned · 0 skipped · 0 parse diagnostics
 next: vanguard explain R2xx-01
```

Quy ước render (bắt buộc, w2-05 code đúng đây):
- Nhóm **theo rule** (chọn này, không theo file): thứ tự nhóm = severity desc (ERROR→WARN→INFO) rồi ruleId asc — PR review sửa từng rule một lần; khớp trật tự `rules[]` SARIF.
- Findings trong nhóm sort `(file, line, col)`.
- Severity pill: TTY = màu (ERROR đỏ, WARN vàng, INFO cyan); non-TTY = text trần `[ERROR]`/`[WARN]`/`[INFO]` — không escape code rác trong CI log.
- Mỗi finding: dòng location `file:line:col`, dòng message, dòng `suggest:` khi có suggestion, dòng `suppress:` **1 lần mỗi nhóm**.
- Footer: tổng kết counts + suppressed + files; dòng `next:` trỏ rule ERROR dày nhất (hoặc nhóm đầu).
- Header line 2: số service/endpoint/grpc + rules active + files + thời gian. `--verbose` thêm: evidence detect, file skipped + lý do, parse diagnostics, config path.
- Màu: auto TTY + `NO_COLOR` + `--no-color` (w1-02 #3). Non-TTY (CI): không màu nhưng vẫn in pretty dạng trần (trừ `check` — §6.1).

### 6.7. Mock output `json`

```json
{
  "vanguard": "0.1.0",
  "schema": 1,
  "target": "./military-youth",
  "source": {"lang": "java", "framework": "spring-boot", "frameworkVersion": "4.0.2"},
  "summary": {
    "files": 598, "skippedFiles": 0, "parseDiagnostics": 0,
    "findings": {"error": 3, "warning": 14, "info": 9},
    "suppressed": 12, "durationMs": 3200
  },
  "findings": [
    {
      "ruleId": "R2xx-01", "slug": "get-no-body", "severity": "ERROR",
      "message": "GET endpoint declares @RequestBody — GET must not have a body.",
      "suggestion": "use @RequestParam for the fields, or switch to POST /auto-complete",
      "location": {"file": "src/main/java/com/viettel/tcct/web/rest/YouthResource.java", "line": 113, "column": 44}
    }
  ],
  "diagnostics": []
}
```

Quy ước: `findings` sort `(file, line, col, ruleId)`; keys sorted; `schema` tăng khi đổi shape; diagnostics = parse errors (best-effort contract §5.3).

### 6.8. Mock `sarif` (2.1.0 — skeleton w2-06/w2-05 code theo)

```json
{
  "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "vanguard",
          "version": "0.1.0",
          "informationUri": "https://github.com/Stellarhold170NT/vanguard",
          "rules": [
            {
              "id": "R2xx-01",
              "name": "get-no-body",
              "shortDescription": {"text": "GET endpoints must not declare @RequestBody"},
              "fullDescription": {"text": "GET endpoints must not declare @RequestBody parameters. Move the payload to @RequestParam or switch to POST."},
              "helpUri": "https://github.com/Stellarhold170NT/vanguard/blob/main/docs/rules/R2xx-01-get-no-body.md",
              "defaultConfiguration": {"level": "error"},
              "properties": {"category": "methods", "aip": "131"}
            }
          ]
        }
      },
      "results": [
        {
          "ruleId": "R2xx-01",
          "ruleIndex": 0,
          "level": "error",
          "message": {"text": "GET endpoint declares @RequestBody — GET must not have a body."},
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {"uri": "src/main/java/com/viettel/tcct/web/rest/YouthResource.java"},
                "region": {"startLine": 113, "startColumn": 44, "endLine": 113, "endColumn": 57, "snippet": {"text": "@RequestBody"}}
              }
            }
          ],
          "partialFingerprints": {"vanguardFindingV1": "<hash(ruleId|uri|startLine|startColumn|message)[:16]>"}
        }
      ],
      "columnKind": "utf16CodeUnits"
    }
  ]
}
```

Quy ước: severity map ERROR→`error`, WARN→`warning`, INFO→`note` (quy ước GitHub); `rules[]` chứa **chỉ các rule có ≥1 result** (full catalog qua `--list-rules`); `uri` relative từ scan root; `partialFingerprints` có từ ngày đầu để GitHub dedup (w2-05); schema validate trong CI (w2-05 acceptance).

### 6.9. Mock `--list-rules` + `explain`

```text
$ vanguard scan --list-rules
ID        SEV    CATEGORY   AIP    SLUG                    SUMMARY
R1xx-01   WARN   resource   131    plural-collection       Collection path uses plural noun
R1xx-02   ERROR  resource   131    no-verb-path            No CRUD verb in path
… (26 rows, sort id) …
26 rules · 25 enabled · 1 disabled by default (R6xx-01) · vanguard 0.1.0
```

```text
$ vanguard explain R2xx-01
R2xx-01 get-no-body · category: methods · severity: ERROR · AIP-131

GET endpoints must not declare @RequestBody parameters.

Bad (Spring):
  @GetMapping("/auto-complete")
  public List<YouthAutocompleteResponse> autoComplete(@RequestBody YouthAutocompleteRequest req) { … }

Good:
  @GetMapping("/auto-complete")
  public List<YouthAutocompleteResponse> autoComplete(@RequestParam String idNumber) { … }

Suppress: // vanguard:ignore R2xx-01 <reason>
Docs: docs/rules/R2xx-01-get-no-body.md
```

(ví dụ good/bad dùng đúng tình huống thật w1-03 pain 7a — YouthResource.java:113-117.)

---

## 7. Chất lượng & gates (R7)

Số liệu dưới là **cam kết sản phẩm**; phương pháp đo chi tiết thuộc `docs/test-strategy.md` (w1-05) — charter chỉ chốt đích + điều kiện:

| Đích | Giá trị | Chứng minh ở đâu |
|---|---|---|
| Recall vs swagger (R1) | **≥95% endpoint** mà swagger/springdoc liệt kê được phát hiện qua scan source, đo trên military-youth (profile `api-docs`) | w5-03 (steel cross-check swagger vs IR) |
| Adversarial corpus | **≥150 case** Java, quy ước `testdata/adversarial/<rule-id>/{ok,vio}-<slug>.java`, mỗi rule ≥3 vio + ≥2 ok, case ≤40 dòng, **không copy code military-youth** (chỉ pattern mô phỏng) | w4-01 |
| Mutation catch-rate | **≥90%** trên mutator bộ (đổi verb, drop pagination, expose entity, rename field…) | w4-02 |
| FP/FN audit | Có số liệu, quy trình label + xử lý tranh chấp (trích aip.dev) | w4-04, số liệu tổng hợp trong GATE report w4-05 |
| Perf & determinism | repo 10k+ file có p95/p99; output byte-identical 2 lần chạy | w4-03 (đích số cụ thể do w1-05 chốt bằng công cụ đo được) |
| Real-fire | military-youth quét thật + FP triage + baseline remediation | w5-01..05, postmortem w5-06 điều chỉnh rule |

Nguyên tắc chống-FP (ràng buộc W3): rule nào heuristic không chắc → severity WARN/INFO + message kèm mời verify — ưu tiên số 1 của W4/W5 là FP, không phải catch-all (w3-04/w3-06 protocol).

---

## 8. Versioning & stability policy

- **Binary**: SemVer. v0.1.0 = release đầu (w6-04, goreleaser w6-01).
- **Rule lifecycle**: thêm/bớt rule = **minor** bump; đổi interface core (IR/adapter/check signature) = **major**. (bám chính sách README api-linter — w1-01 §5.1.9).
- **Stability label**: rule trong v0.1 = `experimental`; sau FP/FN audit W4/W5 + tuning w5-04, rule đạt chuẩn chuyển `stable` ở 0.2. Label nằm trong metadata, hiển thị ở `--list-rules`/`explain`/docs.
- **Charter amendment**: mọi thay đổi charter sau CH1 đi qua orchestrator (issue từ report của task phát hiện), không sửa lén trong task implementation.

---

## 9. Phụ lục A — Quyết định mở chờ CH1 chốt

| # | Quyết định | Khuyến nghị của charter | Hệ quả nếu chọn khác |
|---|---|---|---|
| A-1 | Ngôn ngữ message/CLI: **English** | English (SARIF/GitHub ecosystem, repo công khai, contributor quốc tế; docs tiếng Việt có thể thêm sau) | WARN/INFO text dễ viết tiếng Việt hơn nhưng phá nhất quán output machine-friendly |
| A-2 | Go module path: `github.com/Stellarhold170NT/vanguard` | Theo repo convention (w2-01 cho phép 2 lựa chọn) | `vanguard.io/vanguard` đẹp hơn nhưng cần sở hữu domain — không cần cho OSS |
| A-3 | R6xx-01 `versioned-path` **mặc định TẮT** (opt-in) | Đúng như §3.6 — policy rule, flood WARN trên repo không version | Bật mặc định → lần chạy đầu trên military-youth = 24 WARN vô nghĩa → mất niềm tin first-run |
| A-4 | R1xx-05 `id-field-naming` chỉ **report dùng trộn** (non-prescriptive) trong v0.1 | Đúng — chưa đủ dữ liệu ép `<resource>Id` vs `id` | Ép chuẩn sớm = FP lớn trên DTO legacy |
| A-5 | Envelope thành công (ApiResponse wrapper) **không** có rule prescriptive trong v0.1; R5xx-04 chỉ kiểm error-side consistency | Đúng — pain 5 là thật nhưng wrapping là lựa chọn kiến trúc của app | Ép 1 envelope = FP ồ ạt; để differential mode + config decide sau |
| A-6 | Repo GitHub public/private theo quy ước owner (w1-06) — thông tinUri SARIF dùng URL tuyệt đối | Public nếu owner cho phép (OSS positioning R8) | Private → docs URL trong SARIF/helpUri không public-read được |
| A-7 | WebFlux v0.1 = annotation model (+Mono/Flux unwrap); functional `RouterFunction` → backlog | Đúng | Lint RouterFunction cần CFG trên route DSL — đắt, ít fixture |
| A-8 | Wave-2 (Go/Python) cần mini-recon riêng trước khi viết adapter; backlog đăng ký ở w7-04 | Đúng như §4 | Bắt đầu adapter không recon = lặp lại sai lầm "giả định framework" mà w1-03 đã tránh |

---

## 10. Phụ lục B — Rủi ro kiến trúc

| # | Rủi ro | Tác động | Giảm thiểu |
|---|---|---|---|
| B-1 | tree-sitter binding: cgo chính thức vs pure-Go `smacker` — cross-compile/static build hỏng | Đe dọa R4 (binary tĩnh) | Tiêu chí chốt §5.5; w3-01 quyết + báo cáo; fallback pure-Go |
| B-2 | Thiếu type resolution → heuristic R4xx-01 (entity vs DTO) FP | FP ồ ạt W5, mất uy tín | Dual-signal (persistence annotation OR package pattern config); ERROR chỉ khi tín hiệu mạnh; FP audit W4/W5 |
| B-3 | Meta-annotation (`@CustomGet = @GetMapping`) không resolve | Miss endpoint → recall < 95% | v0.1: known limitation (w3-02) + unresolved log; annotation alias config → backlog |
| B-4 | military-youth licensing/nhạy cảm | Corpus không được chứa code thật (w4-01) | Corpus chỉ pattern mô phỏng; real-fire quét nội bộ W5, không commit code vào repo |
| B-5 | Regex đọc `.proto` (w3-06) giòn với proto phức tạp | Miss/false trên R6xx-02 | Giới hạn scope v0.1: chỉ `service` + `rpc` names; lạ → bỏ qua im lặng + note verbose |
| B-6 | Determinism vỡ do map iteration (bệnh api-linter — w1-01 §5.2.7) | Golden flaky, CI fail giả | Sort mọi iteration; golden harness w2-07 bắt; w4-03 test idempotency |
| B-7 | Perf 10k+ file | Đe dọa w4-03 p95/p99 | Single-pass discovery, size/depth caps, streaming render (w2-05), không giữ cả tree trong RAM |
| B-8 | Rule flood trên legacy repo → first-run UX kém | Adoption chết | Severity kỷ luật §3; R6xx-01 opt-in; config scoping; differential mode backlog (w1-02 #10) |
| B-9 | SARIF schema drift / validate thiếu | SARIF không pass schema (R5) | Pin 2.1.0; validator chạy trong CI (w2-05) |
| B-10 | Scope creep rule org-specific | Vi phạm R8, mất tính generic | Policy §1.3: org-specific chỉ qua config; governance §3.8 |

---

## 11. Phụ lục C — Log mâu thuẫn deps đã xử lý

Protocol w1-04: deps mâu thuẫn → charter ghi quyết định + lý do, không im lặng bỏ qua.

1. **AIP-193/AIP-180 giả định brief ban đầu vs thực tế api-linter (w1-01 §3.1)**: không tồn tại gói rule 193/180 trong api-linter. → Charter thiết kế R5xx/R6xx **mới** từ nguyên lý AIP + thực tế Spring (§3.5–3.6), không tuyên bố "copy từ api-linter". w1-04 ngắn ở đây = R5xx là điểm vanguard dẫn đầu (api-linter không cover vì error payload sống ở runtime).
2. **R1xx-04 `field-naming-consistency` (w3-03) trùng R4xx-02 `field-casing` (w3-05)**: hai brief cùng mô tả DTO field casing. → Quyết: DTO field casing thuộc **R4xx-02** (payload family, AIP-140); R1xx chỉ giữ naming **path/resource** (§3.1). R1xx-05 được dùng cho `id-field-naming`. Engine đăng ký mỗi rule 1 chỗ duy nhất.
3. **w1-03 pain 10 (authz smell) không thuộc taxonomy 6 họ thiết kế API**: recon thấy pattern taxonomy không cover (protocol đã đoán trước tình huống này). → Quyết: không nhét vào v0.1 (org-specific annotation); ghi thành backlog **R6xx-04 `mutation-authorized`** với điều kiện vào (config `authzAnnotations` + FP data W5) — §3.7. Không hardcode org (R8).
4. **w1-03 pain 9 (search 4 kiểu)**: ép convention search trên legacy = FP. → Quyết: không rule search trong v0.1; backlog R1xx-06 `search-convention`, kích hoạt khi có differential mode (§3.7).
5. **Exit code**: w1-01 ghi api-linter chỉ có 2 mức (thua CI scripting), w1-02 chốt chuẩn ngành 0/1/2. → Quyết: **3 mức** (§6.3) — không mâu thuẫn, ghi rõ để w2-04/w2-06 cùng tham chiếu.
6. **Config discovery**: api-linter KHÔNG auto-discover (w1-01 §5.2.5), w1-02 chốt auto-discovery là chuẩn. → Quyết: auto-discovery từ CWD leo lên (§6.4.2) — không học api-linter ở điểm này.
7. **SARIF**: api-linter không có (grep = 0, w1-01 §4.2); w1-02 xác nhận spectral/golangci/protolint đều có. → Quyết: SARIF 2.1.0 từ ngày đầu (§6.8) — lợi thế cạnh tranh, không phải bắt chước.

---

## 12. Phụ lục D — Hợp đồng theo workflow

Mỗi workflow **được phép giả định** những điều sau (nếu thực tế khác → issue, không tự sửa):

- **W2 (w2-01..w2-07)**: layout §5.1; IR inventory §5.2; adapter interface §5.3; rule engine §5.4; config schema §6.4; render mock §6.6–6.8; exit contract §6.3; engine không biết gì về Java (biên kiến trúc w2-04).
- **W3 (w3-01..w3-07)**: taxonomy §3 là danh sách rule (đổi tên slug phải qua charter); heuristic + severity kỷ luật; best-effort contract §5.3; helper library trước rule đầu; registry validation; docs rule từ metadata (w3-07 cross-check `--list-rules`).
- **W4 (w4-01..w4-05)**: đích R7 §7; quy ước testdata; FP là ưu tiên #1; corpus không chứa code military-youth.
- **W5 (w5-01..w5-06)**: military-youth chỉ static-analysis làm chính; real-fire theo AGENT-SETUP-GUIDE (w1-03 §1.5) khi cần response runtime; cross-check swagger dùng springdoc overlay làm chuẩn đối chứng; postmortem điều chỉnh rule theo §8.
- **W6 (w6-01..w6-05)**: goreleaser 3 OS + Docker; release v0.1.0 = 26 rule experimental Java; docs pack từ w3-07 không viết lại.
- **W7 (w7-01..w7-05)**: wave-2 backlog đăng ký tại w7-04 theo điều kiện §4; ratchet dựa trên golden + audit W4/W5.

---

## 13. Phụ lục E — Backlog v0.1.1 (đề xuất từ postmortem w5-06, chờ gate phê duyệt)

> Sinh từ postmortem GATE CH4 (`reports/w5-06.md`, task #31). Đây là **danh mục đề xuất có thứ tự ưu tiên** — input hợp đồng cho chu kỳ W3 kế tiếp; **chưa có mục nào được thi hành** trong task w5-06. Mỗi mục mang id `ADJ-xx` để truy vết qua các task của cycle W3 sau này. Tiêu chí nghiệm thu đo được (QC Φ1–Φ5) và toàn bộ ngữ cảnh/lead nguồn: `reports/w5-06.md` §6.

| Prio | ID | Loại | Mục | Rule(s) | Effort |
|---|---|---|---|---|---|
| P1 | ADJ-01 | fix-rule (adapter+rule) | Adapter capture `ResponseEntity.created(...)` → IR `Response.StatusCode`=201 (cluster C1a — vắng diện gây 14 FP giả) | R2xx-02, R5xx-03 | M |
| P1 | ADJ-02 | fix-rule | Predicate create-shape đòi **tín hiệu create** (tên method `create*/save*/register*` hoặc token path); mở rộng action lexicon (filter, approval, recall, reject, send, split, merge, dissolve, permission, generate, sync, submit) — mỗi entry lexicon mới phải có fixture (kỷ luật w4-04 §6.2) và sync `internal/mutation/mutators.go` (C1b) | R2xx-02, R5xx-03 | M |
| P1 | ADJ-03 | fix-rule | Phân vai segment (collection noun vs action vs base path) khi kiểm số nhiều; thống nhất base-segment checking giữa các method (C3, FN-4/FN-5) | R1xx-01 | M |
| P1 | ADJ-04 | config + rule metadata | Thêm key `defaultDisabled` cho demo rules (mở rộng schema config v1 — cần duyệt, w5-04 §4 C4); tạm bù bằng config disable T-2 | R6xx-92/93 | S |
| P2 | ADJ-05 | new-rule (đăng ký engine) | Đăng ký 4 rule charter-chưa-đăng-ký: R3xx-04, R4xx-04, R5xx-04, R6xx-03 (declared gaps dm-01..04; mutation strict 85.7%→dự báo 100% khi pass) | — | L |
| P2 | ADJ-06 | fix-rule | **Verify-trước-rồi-sửa** package-resolution trên direct response TypeRef (FN-1/2/3) — IR dump + regression fixture trước khi đụng rule | R4xx-01 | M |
| P1 | ADJ-07 | fix-rule (redesign) | R2xx-05: chỉ fire khi có tín hiệu side effect (**uni của path-verb + method-name verb** — khử blindness FN-6), hoặc đổi message thành verify-hint điều kiện (C2) | R2xx-05 | M |
| P2 | ADJ-08 | process (orchestrator) | Merge `agent-w4-05` (glob-fix) vào main **trước W6**; sau merge tái chạy dry-run T-1: kỳ vọng 244→232 (w5-04 §6 T-1 / §10.2) | — | S |
| P2 | ADJ-09 | fix-rule (adapter) | gRPC: verify + khử gap IR — 3 service shell/0 rpc vs 11 service/88 rpc đếm raw w1-03 (w5-03 §5.3) | R6xx-02 | M |
| P3 | ADJ-10 | corpus | Mở rộng corpus theo w4-02 §7: bare-create ok-case cho R2xx-02 (mở pool remove-status-mapping 1→≥2), GET-single ok-case cho R2xx-01 | — | M |
| P3 | ADJ-11 | investigation | Tra không-match 54-vs-27 parse diagnostics (report vs discovery layer; w5-03 §5.4) — cho đến khi giải quyết xong: gắn cờ traceability risk khi đọc diagnostics | — | S |
| ✓ done | ADJ-12 | data-integrity | (ĐÃ THỰC HIỆN trong w5-06) Vá 1 dòng prose `reports/w5-04.md` §2 (WARN 75/68 → **64/79**, INFO 31/46 → **42/35**; tổng 129/115 không đổi) theo đối chiếu 3 nguồn `reports/w5-05.md` §3 | — | S |

Protocol cho backlog này: sau phê duyệt CH4, mục nào bị bác sẽ ghi rationale vào mục tương ứng của `reports/w5-06.md` (giữ Audit log); thay đổi ngoài danh mục này = phát sinh → đi qua issue của task phát hiện (§3.8).

---

*Hết charter v1. Mọi sửa đổi sau CH1 phải có entry trong Phụ lục C (log quyết định) — charter sống bằng tính truy vết, không bằng trí nhớ.*
