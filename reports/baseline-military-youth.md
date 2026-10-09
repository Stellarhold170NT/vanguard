# Baseline chất lượng API — backend military-youth

**Ngày đo**: 2026-10-09 · **Đối tượng**: source code military-youth (Spring Boot 4.0.2, nhánh `develop`, 774 file) · **Công cụ đo**: `vanguard` scan tĩnh @ commit `f2fc550` · **Người lập**: agent w5-05 (chương trình VANGUARD) · **Trạng thái**: chờ team xác nhận scope sửa + gate ratify nhãn FP

> Tài liệu này viết cho team military-youth. Không cần biết gì về công cụ đo để đọc hiểu;
> mọi con số đều truy vết được về file artifact (Phụ lục C). Phần 1–3 + phần 5 đọc trong ~5 phút.

---

## 1. Tóm tắt điều hành (đọc 5 phút)

Backend military-youth có **182 REST endpoint** (24 controller, `src/main/java`). Một đợt quét tĩnh
toàn bộ source bằng bộ ~27 quy tắc thiết kế API (chuẩn AIP — Google API Improvement Guidelines)
phát hiện **244 điểm cần xem lại**, sau khi xét từng điểm trên source code thật còn lại **129 vấn đề
thật** (115 còn lại là báo nhầm của tool, chi tiết phần 6).

| Con số | Giá trị | Ý nghĩa cho team |
|---|---|---|
| REST endpoint phát hiện | **182** unique (method+path) · 184 handler · 24 controller | full surface, so khớp 1:1 với annotation trong code — **không có endpoint nào bị bỏ sót** (recall 182/182 = 100%, ngưỡng chương trình ≥ 95%) |
| Findings thô | **244** = 24 ERROR · 143 WARN · 77 INFO | trước khi phân loại người/thật |
| Sau triage | **129 vấn đề thật (TP)** · 115 báo nhầm (FP) | FP là do heuristic của tool, **không phải lỗi của code** |
| Đáng tin nhất | **23/24 finding ERROR là thật (95.8%)** | mọi finding ERROR đều đáng xử lý, trừ 1 exception |
| Bị bỏ sót | **6 nghi vấn (FN)** — đọc tay, chưa đầy đủ | 3 trong số đó là code của team nên sửa luôn (mục B1) |

**Ba nhóm vấn đề thật chiếm phần lớn khối lượng sửa:**

1. **8 endpoint trả thẳng ORM entity** ra HTTP response (ERROR) — rò schema DB ra ngoài, vỡ khi entity đổi.
2. **~21 endpoint có shape route phi chuẩn**: động từ CRUD trong path (`/delete`, `/find-by-id`) và/hoặc
   DELETE khai request body (ERROR/WARN) — cần redesign route, ảnh hưởng FE.
3. **13 endpoint create trả 200 thay vì 201** (WARN) — sửa nhanh, pattern đã có sẵn trong chính codebase.

Kèm theo là **backlog 10 mục** (phần 5) xếp theo ưu tiên ERROR → WARN → INFO, mỗi mục có hiện trạng,
đề xuất sửa, quy tắc liên quan và vị trí file:dòng chính xác.

---

## 2. Phạm vi và cách đo (3 dòng)

- Quét tĩnh toàn repo military-youth (`vanguard scan`, binary `f2fc550`, 2026-10-09, ~4 giây/lần,
  753/774 file đọc được — 9 file bị skip, 54 parse-diagnostics chưa phân loại hết, xem Phụ lục A.3).
- Không có cấu hình `.vanguard.yaml` nào trong target → quy tắc mặc định (27 rule nạp, 18 rule bắn,
  **0 suppression áp dụng** — mọi finding đều từ engine nguyên bản).
- Recall phát hiện endpoint đối chiếu với danh mục endpoint trích độc lập từ annotation Spring:
  **182/182 = 100%** (giới hạn phương pháp: so khớp 2 parser tĩnh, không chạy được swagger runtime —
  Phụ lục A.4).

---

## 3. Kết quả chi tiết

### 3.1. Findings theo severity × verdict

| Severity | Findings | TP (vấn đề thật) | FP (báo nhầm) | Precision |
|---|---:|---:|---:|---:|
| ERROR | 24 | 23 | 1 | **95.8%** |
| WARN | 143 | 64 | 79 | 44.8% |
| INFO | 77 | 42 | 35 | 54.5% |
| **Tổng** | **244** | **129** | **115** | **52.9%** |

Đọc bảng này như người lạ: ERROR = tool gần như không bao giờ nhầm (23/24); WARN/INFO cần lọc
(tool nhầm ~1/2), và phần nhầm đã được xét từng dòng — chỉ phần TP là việc của team.

### 3.2. Từng quy tắc (18 rule đã bắn)

| Rule | Tên | Sev | Findings | TP | FP | FN nghi vấn |
|---|---|---|---:|---:|---:|---:|
| R1xx-01 | path collection dùng danh từ số ít | WARN | 21 | 2 | 19 | 2 |
| R1xx-02 | động từ CRUD trong path | ERROR | 14 | 14 | 0 | 0 |
| R1xx-03 | `{id}` nằm dưới action segment | WARN | 2 | 2 | 0 | 0 |
| R1xx-04 | path không kebab-case | WARN | 1 | 1 | 0 | 0 |
| R2xx-01 | GET khai request body | ERROR | 2 | 1 | 1 | 0 |
| R2xx-02 | POST create trả 200 thay vì 201 | WARN | 30 | 12 | 18 | 0 |
| R2xx-03 | PATCH nhận full body | WARN | 4 | 0 | 4 | 0 |
| R2xx-04 | DELETE khai request body | WARN | 14 | 14 | 0 | 0 |
| R2xx-05 | action đặt nhầm verb (GET/PUT có side effect) | INFO | 34 | 4 | 30 | 1 |
| R2xx-06 | PUT làm partial update | WARN | 7 | 1 | 6 | 0 |
| R3xx-01 | list GET không phân trang | WARN | 20 | 20 | 0 | 0 |
| R3xx-02 | list trả mảng trần không envelope | INFO | 27 | 24 | 3 | 0 |
| R4xx-01 | ORM entity lộ trong payload | ERROR | 8 | 8 | 0 | 3 |
| R4xx-03 | field thời gian không RFC3339 | INFO | 11 | 11 | 0 | 0 |
| R5xx-03 | status code không đúng ngữ nghĩa | WARN | 32 | 12 | 20 | 0 |
| R6xx-02 | rpc proto không đặt tên Standard Method | INFO | 3 | 3 | 0 | 0 |
| R6xx-92 | demo rule — method không khai response type | WARN | 12 | 0 | 12 | 0 |
| R6xx-93 | demo rule — GET khai body | INFO | 2 | 0 | 2 | 0 |
| **Tổng** | | | **244** | **129** | **115** | **6** |

Ghi chú: R6xx-92/93 là **rule demo** đi kèm tool, không thuộc bộ quy tắc chính thức — 14 findings của
chúng đều là FP, sẽ được tắt bằng config (mục 6). FN nghi vấn = vi phạm thật mà tool không bắt,
phát hiện bằng đọc tay có giới hạn scope (Phụ lục A.4.2).

### 3.3. Hotspot theo file (TP)

| File | TP | File | TP |
|---|---:|---|---:|
| `web/rest/YouthUnionResource.java` | 12 | `web/rest/UnionMembershipExpenditureResource.java` | 6 |
| `web/rest/YouthResource.java` | 10 | `web/rest/MagazineArticlesResource.java` | 5 |
| `web/rest/CategoryDataResource.java` | 8 | `web/rest/MemberEvaluationResource.java` | 5 |
| `web/rest/TrainingPlanResource.java` | 7 | `web/rest/InnovationByUnitResource.java` | 5 |
| `web/rest/UnionEvaluationResource.java` | 7 | `web/rest/YouthInnovationRegistrationResource.java` | 5 |
| `web/rest/UnionMembershipCollectionResource.java` | 7 | `web/rest/MovementReportResource.java` | 7 |

31/35 file có findings cũng có ít nhất 1 TP; danh sách đủ 129 dòng ở Phụ lục B.

---

## 4. Danh mục 129 vấn đề thật (theo nhóm quy tắc)

Bảng đầy đủ từng dòng (file:dòng, message gốc của scan) ở **Phụ lục B**. Nhóm chính:

- **22 span ERROR**: 8 ORM-entity leak (R4xx-01) + 14 động-từ-trong-path (R1xx-02) + 1 GET-body
  (R2xx-01) — vài span vi phạm cùng lúc 2 quy tắc.
- **64 span WARN**: 12+12 cặp create-200 (R2xx-02 + R5xx-03 cùng span, 13 endpoint thật sau gộp),
  14 DELETE-body (R2xx-04), 20 list không phân trang (R3xx-01), 1 PUT-partial (R2xx-06),
  2 `{id}`-dưới-action (R1xx-03), 1 casing (R1xx-04), 2 base-path số ít (R1xx-01).
- **42 span INFO**: 24 mảng trần (R3xx-02), 11 field thời gian (R4xx-03), 4 action sai verb
  (R2xx-05), 3 rpc proto (R6xx-02).

## 5. Remediation backlog — việc cho team military-youth

Sắp theo **priority** (P1 = ERROR, sửa trước; P2 = WARN; P3 = INFO) và **effort ước tính**
(S ≤ 0.5 ngày · M ≈ 1–3 ngày · L > 3 ngày, gồm cả chỉnh FE + test — ước lượng của người lập,
team tự hiệu chỉnh). Mỗi mục: hiện trạng → đề xuất → quy tắc → vị trí.

---

### B1 · P1 · ERROR — ORM entity lộ ra HTTP response (8 span)

- **Hiện trạng**: 8 endpoint trả trực tiếp JPA entity (`Youth`, `YouthUnion`, `UnionEvaluation`,
  `MagazineArticles`…) trong body — schema DB, quan hệ lazy, field nội bộ leak ra wire; mỗi lần
  entity đổi là vỡ contract client ngoài ý muốn.
- **Đề xuất**: map sang DTO qua mapper trước khi trả; riêng 2 endpoint `/import` trả `ImportResult<…>`
  cần đổi cả kiểu wrap. Làm luôn 3 span cùng dạng mà tool chưa bắt (w5-04 FN-1..3):
  `YouthResource.java:~121` (GET `/current`), `MemberEvaluationResource.java:~139` (POST create),
  `MemberEvaluationResource.java:~158` (PUT `/{id}`).
- **Quy tắc**: R4xx-01 no-entity-in-payload (aip.dev/121).
- **Vị trí**: `MagazineArticlesResource.java:56` · `UnionEvaluationResource.java:116` ·
  `UnionEvaluationResource.java:150` · `UnionMembershipCollectionResource.java:231` ·
  `YouthInnovationRegistrationResource.java:285` · `YouthResource.java:86` ·
  `YouthUnionResource.java:205` · `YouthUnionResource.java:258`
- **Effort**: M (8 span + 3 span bổ sung; test hợp đồng response).

### B2 · P1 · ERROR — Route phi chuẩn: động từ CRUD trong path + DELETE kèm body (~21 endpoint)

- **Hiện trạng**: 14 endpoint nhét động từ vào path (`/delete`, `/update`, `/get-term`,
  `/find-by-id`, `/find-all-id-active`…) — HTTP verb đã đủ nghĩa; 14 DELETE khai
  `@RequestBody List<String>` (bulk delete qua body — nhiều HTTP client/proxy không hỗ trợ);
  7 endpoint vi phạm cả hai cùng lúc.
- **Đề xuất**: mỗi resource về bộ chuẩn: `GET/PUT/PATCH/DELETE /{collection}/{id}`; bulk delete →
  `POST /{collection}:batchDelete` (hoặc query param); xoá segment động từ. Vì là thay đổi hợp đồng
  API: làm theo resource, giữ endpoint cũ ở 1–2 release (deprecated) để FE chuyển dần.
- **Quy tắc**: R1xx-02 no-verb-path (aip.dev/131) · R2xx-04 delete-no-body (aip.dev/135, bulk →
  aip.dev/165).
- **Vị trí** (R1xx-02, 14): `DefenseDiplomacyResource.java:217` · `MagazineListQuarterlyResource.java:227`
  · `MovementReportResource.java:135,146` · `ReportResource.java:161` · `TrainingPlanResource.java:215,221`
  · `TrainingResultResource.java:148` · `UnionMembershipCollectionResource.java:211` ·
  `UnionMembershipExpenditureResource.java:206` · `YouthInnovationRegistrationResource.java:226` ·
  `YouthMovementGuidelinesResource.java:193` · `YouthUnionResource.java:258,382`.
  (R2xx-04, 14: danh sách ở Phụ lục B, nhóm R2xx-04 — trùng 7 span với trên.)
- **Effort**: L (per resource; gom với B4/B6 khi sửa cùng controller).

### B3 · P1 · ERROR — GET `/youths/auto-complete` khai request body (1 span)

- **Hiện trạng**: GET có `@RequestBody YouthAutocompleteRequest` — nhiều HTTP client (một số
  WebClient/OkHttp/curl mặc định) bỏ body của GET → chức năng hỏng âm thầm tuỳ client.
- **Đề xuất**: chuyển tham số sang query string (`?keyword=&limit=`); bỏ `@RequestBody`.
- **Quy tắc**: R2xx-01 get-no-body (aip.dev/131).
- **Vị trí**: `YouthResource.java:116`.
- **Effort**: S.

### B4 · P2 · WARN — 13 endpoint create trả 200 thay vì 201

- **Hiện trạng**: POST tạo resource trả `ResponseEntity.ok(...)` (implicit 200). Chính codebase đã có
  pattern đúng ở 7 controller khác (`ResponseEntity.created(new URI(...))`) — thiếu nhất quán.
- **Đề xuất**: đổi sang `ResponseEntity.created(location).body(dto)`; endpoint draft của
  `MovementReportResource` (`/draft`, `/result/draft`) quyết định rõ: nếu tạo resource mới → 201,
  nếu chỉ update bản nháp có sẵn → 200 (giữ nguyên là đúng).
- **Quy tắc**: R2xx-02 post-creates-201 + R5xx-03 status-semantics (aip.dev/133).
- **Vị trí** (13, sau gộp 2 rule): `DocumentResource.java:49` · `InnovationByUnitResource.java:51` ·
  `MemberEvaluationResource.java:139` · `MovementReportResource.java:37,48,93` · `ReportResource.java:53`
  · `SystemCodeConfigResource.java:38` · `UnionEvaluationResource.java:116` ·
  `UnionMembershipCollectionResource.java:71` · `UnionMembershipExpenditureResource.java:66` ·
  `YouthInnovationRegistrationResource.java:73` · `YouthResource.java:58`.
- **Effort**: S–M (mechanical; FE cần biết 201 mới là success-code mới).

### B5 · P2 · WARN — 20 list GET không phân trang

- **Hiện trạng**: 20 GET trả list full không tham số phân trang — rủi ro dump khi dữ liệu lớn
  (auto-complete, search, tree, terms…).
- **Đề xuất**: thêm phân trang cho list dữ liệu lớn (page/size hoặc cursor); **trừ nhóm danh mục
  bounded** (provinces/wards/countries/dropdown của `CategoryDataResource`, `fee-terms`,
  `assessment-terms`, `get-term`) — nếu team quyết định các list này cố tình full (dữ liệu nhỏ,
  dùng cho dropdown) thì ghi convention và xử lý bằng suppression có lý do (mục 6), không sửa code.
- **Quy tắc**: R3xx-01 list-paginated (aip.dev/158).
- **Vị trí**: 20 span — `CategoryDataResource.java:36,42,50,55` · `CitizenResource.java:36,45` ·
  `DepartmentResource.java:28,37` · `MagazineArticlesResource.java:56` · `MemberEvaluationResource.java:123`
  · `TrainingPlanResource.java:215` · `TrainingResultResource.java:148` · `UnionEvaluationResource.java:108`
  · `UnionMembershipCollectionResource.java:244` · `UnionMembershipExpenditureResource.java:232` ·
  `YouthResource.java:115,129` · `YouthUnionResource.java:238,258,367`.
- **Effort**: M (số lớn: cần quyết design; số nhỏ: quyết policy là xong).

### B6 · P2 · WARN — Sửa lẻ shape route (4 span)

- **Hiện trạng + đề xuất**:
  1. `YouthResource.java:65` — `partialUpdateYouth` dùng **PUT** cho partial update → đổi thành
     **PATCH** (PUT để dành cho full-replace). Rule R2xx-06 (aip.dev/134).
  2. `YouthMovementGuidelinesResource.java:193` + `YouthUnionResource.java:382` — `{id}` nằm dưới
     action segment (`/update-status-uncheck/{id}`, `/find-by-id/{id}`) → đưa về `/{collection}/{id}`
     (tự gộp khi làm B2/B7). Rule R1xx-03 (aip.dev/127).
  3. `MagazineArticlesResource.java:56` — segment `all-by-Name` hoa giữa kebab → `all-by-name`
     (7 chỗ khác trong repo đã đúng). Rule R1xx-04 (aip.dev/122).
- **Effort**: S mỗi span.

### B7 · P3 · INFO — 4 action có side effect đang dùng GET/PUT

- **Hiện trạng**: thao tác đổi trạng thái đang đi bằng GET/PUT: sync-departments (GET),
  approve ×2 (PUT), update-status-uncheck (PUT) — GET có side effect bị cache/proxy/retry hành
  xử sai.
- **Đề xuất**: chuyển sang POST (AIP-136: method có side effect phải dùng POST).
- **Quy tắc**: R2xx-05 custom-method-post (aip.dev/136).
- **Vị trí**: `VauthzController.java:44` (GET `/auth/sync-departments`) ·
  `TrainingPlanResource.java:240` (PUT `/training-plans/approve`) ·
  `TrainingResultResource.java:100` (PUT `/training-results/approve`) ·
  `YouthMovementGuidelinesResource.java:193` (PUT `/update-status-uncheck/{id}`).
- **Effort**: S.

### B8 · P3 · INFO — 24 list trả mảng JSON trần

- **Hiện trạng**: response là `List<…>` trần — không có chỗ cho metadata (total, page) và khó version.
  Trùng 20 span với B5; 4 span riêng: `DefenseDiplomacyResource.java:170`,
  `MagazineArticlesResource.java:158`, `MagazineListQuarterlyResource.java:173`,
  `TrainingPlanResource.java:171`.
- **Đề xuất**: chọn 1 envelope cho toàn backend (ví dụ `{data, page, total}`) và áp dụng cùng lúc
  với B5; nếu team theo convention mảng trần thì ghi convention + suppression có lý do.
- **Quy tắc**: R3xx-02 list-envelope (aip.dev/158).
- **Effort**: M (quyết 1 lần, áp theo controller).

### B9 · P3 · INFO — 11 field thời gian dùng String/LocalDateTime

- **Hiện trạng**: các DTO trả `fromDate`/`toDate`/`rewardDate`/`publishDate`… bằng `String` hoặc
  `LocalDateTime` — client không parse được múi giờ (thiếu offset/RFC3339).
- **Đề xuất**: đổi sang `Instant`/`OffsetDateTime` (serialize RFC3339); các field `display*` nếu giữ
  String định nghĩa rõ format và document.
- **Quy tắc**: R4xx-03 time-field-standard (aip.dev/142).
- **Vị trí**: `UnionMemberHistoryDTO.java:20,21` · `YouthInnovationRegistrationDTO.java:60` ·
  `DefenseDiplomacyResponse.java:24,25,26,27` · `MagazineListQuarterlyResponse.java:23` ·
  `TrainingPlanResponse.java:23,24` · `YouthInnovationRegistrationResponse.java:74`.
- **Effort**: M (11 field, test FE parse).

### B10 · P3 · INFO — 3 rpc proto đặt tên không chuẩn (R6xx-02)

- **Hiện trạng**: `PartialUpdateDefenseDiplomacy` / `PartialUpdateDefenseParticipants` /
  `PartialUpdateDefenseSchedule` — tên rpc ngoài bộ Standard Method (Get*/List*/Create*/Update*/Delete*).
- **Đề xuất**: đổi thành `Update*` nếu semantics là update; giữ kèm ghi chú mapping nếu cần tương thích.
- **Quy tắc**: R6xx-02 grpc-standard-methods (aip.dev/134).
- **Vị trí**: `src/main/proto/entity/defense_diplomacy.proto:17` · `defense_participants.proto:16` ·
  `defense_schedule.proto:17`.
- **Effort**: S.

---

## 6. Phần KHÔNG phải việc sửa code của team (bối cảnh cần biết)

- **115 báo nhầm (FP)**: cụm lớn nhất là heuristic của tool chưa đọc được runtime status
  (`ResponseEntity.created`) và chưa phân biệt action vs create (38 FP), GET thuần đọc (30 FP),
  segment hành động trong path (19 FP), rule demo (14 FP), còn lại 14 FP rải. Các cụm này đã có
  phương án chỉnh engine (thuộc w5-06 — do chủ dự án tool xử lý, không phải team military-youth).
- **Trong lúc chờ engine sửa**: nếu finding FP làm khóReview, team có thể suppress có kiểm soát
  (inline comment `// vanguard:ignore <rule> <lý do>`) theo mẫu đã soạn sẵn trong
  `reports/w5-04.md` §7.3 — nguyên tắc: chỉ suppress finding đã có verdict FP, luôn kèm lý do,
  không suppress-before-review để CI xanh.
- **2 rule demo R6xx-92/93** (14 FP) sẽ được tắt bằng config — đang chờ quyết định gate, chưa áp.
- **6 FN nghi vọng** — 3 mục là code team nên sửa luôn (đã gộp vào B1); 3 mục còn lại là giới hạn
  của engine (base-path số ít `department`/`category-data`, GET `/routes` có side effect) — không
  khẩn cấp, đã ghi cho w5-06.

## 7. Có thể tái lập thế nào (tóm tắt — chi tiết Phụ lục A)

```bash
git clone https://github.com/Stellarhold170NT/vanguard.git && git checkout f2fc550 && make bin
./bin/vanguard scan /path/to/military-youth --format json    # exit 1 = có ≥1 finding ERROR (đúng thiết kế)
```

Kết quả phải là 244 findings (24E/143W/77I), byte-identical giữa các lần chạy (sha256
`c4d70013…` của artifact chuẩn). Suppression áp dụng lúc đo: **0**.

---

## Phụ lục A — Phương pháp, cấu hình, giới hạn (để tái lập)

### A.1. Môi trường & phiên bản

| Thành phần | Giá trị |
|---|---|
| Tool | `vanguard` @ commit `f2fc550d88164e43d9083ac5c76adc0ef4faafc8` (nhánh main), build 2026-10-09T16:18:21Z, binary 8,852,176 B, sha256 `56c657bc8dcb62c031baae7e52c4043058ac819867f738bba6064d0a8dff900d`, go1.26.8 linux/amd64 |
| Target | military-youth, VFS checkout `mil-svc-youth` (nhánh `develop` theo ingest; checkout không có `.git`) — 774 file, 3.6 MB, content fingerprint `ced5aa22df0f1b2e03967f63eff59baaf25bf28257a20a3b991063c784aeab23` (sha256 multiset các per-file sha256) |
| Framework | java · spring-boot 4.0.2 (auto-detect) |
| Thời gian đo | 2026-10-09 16:09–16:25 UTC (scan), recall 16:33–17:05 UTC, triage 16:34–17:4x UTC |
| Cách chạy | sandbox `dind-sandbox:29` (Alpine, 2 CPU / 4 GB) — môi trường đo được cô lập khỏi workspace |

### A.2. Lệnh & cấu hình scan

- Lệnh: `./bin/vanguard scan /workspace --format json` (cùng lệnh với `pretty`/`sarif` cho 2 format
  còn lại); ~4 giây/lần; exit code 1 = "có ≥1 finding ERROR còn sống" (hợp đồng của tool, không phải lỗi).
- Cấu hình: **không có** `.vanguard.yaml` trong target (đếm = 0) → default: 27 rule nạp
  (18 bắn), include `**`, không exclude, không suppression. `summary.suppressed = 0`.
- Idempotency: 2 lần chạy json độc lập cho output **byte-identical**; 3 format cùng phiên cho cùng
  244/24/143/77.
- Diagnostics: 54 parse-diagnostics (1 file không parse được: `DocumentRequest.java` ×2; phần còn
  lại chủ yếu unresolved model-attribute) — giữ nguyên trong `scan.json`, chưa phân loại hết.

### A.3. Số liệu phát hiện endpoint (recall)

- Tham chiếu: trích **độc lập** từ annotation Spring MVC (`@RestController` + mapping) bằng parser
  Python riêng (`reports/w5-03-refparse.py`) — 27 controller class, 24 có endpoint, 184 handler →
  **182 endpoint unique** (3 overload cùng `GET /youth-unions/exists` gộp 1 ở cấp swagger).
- Đối chiếu: IR của tool (dump `discovery.Scan()` @ cùng commit) — 184 node `src/main` khớp từng
  verb (GET 87 · POST 54 · DELETE 20 · PUT 19 · PATCH 4).
- Kết quả: matched 182 · FN 0 · thừa 0 → recall 100.0% (script `reports/w5-03-compare.py`,
  output `artifacts/w5-03/compare.json`). Bảng 182 dòng hai phía: `reports/w5-03-endpoints-table.md`.

### A.4. Giới hạn phương pháp (đọc trước khi viện dẫn con số)

1. **Swagger runtime không chạy được**: app cần 7 jar riêng tư + 2 DB + Redis + ~15 env, springdoc
   bị gate sau profile `api-docs`, và không tồn tại deployment nào đã biết → recall là **đồng thuận
   của 2 parser tĩnh** cùng đọc source annotation (cận trên của recall-so-swagger-thật). Các kênh
   endpoint mà chỉ runtime thấy (RouterFunction, actuator) đã kiểm: không có trong target.
2. **FN (6 nghi vấn) không phải phép đo đầy đủ**: là kết quả đọc tay 24 controller/DTO có finding +
   targeted grep — không phải sweep toàn corpus; phép đo FN hệ thống thuộc mutation harness (w4-02).
3. **Nhãn TP/FP do agent w5-04 gắn, CHỜ gate CH3 ratify** — nếu gate đổi nhãn, số phần 3 thay đổi
   theo; phép đếm máy (244/18-rule/severity) không đổi.
4. **54 parse-diagnostics + 9 file skip chưa phân loại hoàn toàn** — về nguyên tắc có thể che mất
   một số finding; không ảnh hưởng kết quả recall (file lỗi không phải file endpoint).
5. **gRPC ngoài phép đo recall**: IR tool chỉ có 3 service shells / 0 rpc (raw: 11 proto / 88 rpc)
   — gap của engine, đã ghi cho w5-06; 3 finding R6xx-02 trong baseline là phần proto tool thấy được.
6. **11 endpoint `src/test`** xuất hiện trong IR nhưng bị loại khỏi recall (runtime scope) — 12
   findings trên test fixture đã triage FP.
7. **Số dòng `~n`** trong một số vị trí (ví dụ B1) = ước lượng từ đọc tay của w5-04 (FN), không phải
   số dòng chính xác như các vị trí khác (số chính xác đến từ scan.json).

### A.5. Artifact gốc (sha256 pin)

| Artifact | Nội dung | Bytes | sha256 |
|---|---|---:|---|
| `artifacts/w5-02/scan.json` | 244 findings + summary + diagnostics (chuẩn so khớp) | 148,584 | `c4d7001310d68efbfecd2560c67b0a334a466e15a0911e2a1bcb1aec55791934` |
| `artifacts/w5-02/scan.pretty.txt` | bản pretty cùng lần scan | 83,369 | `8680c32eda0fd9ba0b3b49c19407fc29d876a8be4d5b22db5bc444c97a2ccbc8` |
| `artifacts/w5-02/scan.sarif` | bản SARIF (CI) | 212,133 | `ad7e518e60a2fb254951b8e97e834241db740f33d703f7e66f8ee6b8f80e0500` |
| `artifacts/w5-03/ir-f2fc550.json` | API surface (IR) của tool | 907,467 | `8f6858e8f79f4108f7a28a015cae3a547c40038ba57a8d89206b071c5cedc51c` |
| `artifacts/w5-03/swagger-ref.json` | tham chiếu 184 endpoint từ annotation | 44,724 | `12358dd9563f76b8d5363114ab1b7c04605c4bb24a5ad73894f6cca1b500b6a4` |
| `artifacts/w5-03/compare.json` | kết quả recall máy-đọc | 17,990 | `2bfd04ead8ec39b8188a653ee53b95420f51d7c58c332a8cd826f9604daf36cb` |
| `reports/w5-03-endpoints-table.md` | bảng 182 endpoint hai phía | 39,711 | `3f62f6ade120ddab7dea0a7f77a7e4271f95a4f07b0360fd020d29c8a3d63b48` |
| `testdata/audit-results.json` @ `agent-w5-04` | verdict máy-đọc 244 findings | — | `d23f351345638882477dac47b4f6b4d3db214ec79047bd406af7c81686a7d448` |
| `reports/w5-04.md` | triage sheet 244 dòng + tuning/suppression đề xuất | — | (file workspace, xem chương trình) |

Đường dẫn artifact tính từ workspace chương trình (`/app/workspace/vanguard/`).

## Phụ lục B — 129 vấn đề thật (TP) — từng dòng

Nguồn: sinh tự động từ `scan.json` (vị trí + message nguyên bản) × mapping verdict w5-04
(script tái-đếm fail-closed, chạy lại 2026-10-09: 31/31 checks PASS). Message là nguyên văn tiếng
Anh của scan, rút đúng bản gốc.

| # | Rule | Sev | Vị trí (repo military-youth) | Message (nguyên bản scan) |
|---|---|---|---|---|
| 0 | R4xx-03 | INFO | `src/main/java/com/viettel/tcct/service/dto/UnionMemberHistoryDTO.java:20` | Field "fromDate" holds a time value as LocalDateTime — clients get no RFC3339 structure to parse. |
| 1 | R4xx-03 | INFO | `src/main/java/com/viettel/tcct/service/dto/UnionMemberHistoryDTO.java:21` | Field "toDate" holds a time value as LocalDateTime — clients get no RFC3339 structure to parse. |
| 2 | R4xx-03 | INFO | `src/main/java/com/viettel/tcct/service/dto/YouthInnovationRegistrationDTO.java:60` | Field "rewardDate" holds a time value as LocalDateTime — clients get no RFC3339 structure to parse. |
| 3 | R4xx-03 | INFO | `src/main/java/com/viettel/tcct/service/dto/response/DefenseDiplomacyResponse.java:24` | Field "fromDate" holds a time value as String — clients get no RFC3339 structure to parse. |
| 4 | R4xx-03 | INFO | `src/main/java/com/viettel/tcct/service/dto/response/DefenseDiplomacyResponse.java:25` | Field "displayFromDate" holds a time value as String — clients get no RFC3339 structure to parse. |
| 5 | R4xx-03 | INFO | `src/main/java/com/viettel/tcct/service/dto/response/DefenseDiplomacyResponse.java:26` | Field "toDate" holds a time value as String — clients get no RFC3339 structure to parse. |
| 6 | R4xx-03 | INFO | `src/main/java/com/viettel/tcct/service/dto/response/DefenseDiplomacyResponse.java:27` | Field "displayToDate" holds a time value as String — clients get no RFC3339 structure to parse. |
| 7 | R4xx-03 | INFO | `src/main/java/com/viettel/tcct/service/dto/response/MagazineListQuarterlyResponse.java:23` | Field "publishDate" holds a time value as String — clients get no RFC3339 structure to parse. |
| 8 | R4xx-03 | INFO | `src/main/java/com/viettel/tcct/service/dto/response/TrainingPlanResponse.java:23` | Field "startDate" holds a time value as String — clients get no RFC3339 structure to parse. |
| 9 | R4xx-03 | INFO | `src/main/java/com/viettel/tcct/service/dto/response/TrainingPlanResponse.java:24` | Field "endDate" holds a time value as String — clients get no RFC3339 structure to parse. |
| 10 | R4xx-03 | INFO | `src/main/java/com/viettel/tcct/service/dto/response/YouthInnovationRegistrationResponse.java:74` | Field "rewardDate" holds a time value as LocalDateTime — clients get no RFC3339 structure to parse. |
| 11 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/CategoryDataResource.java:36` | GET /api/military-youth/category-data/provinces returns an unpaginated collection — one client request can dump the whole table. |
| 13 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/CategoryDataResource.java:42` | GET /api/military-youth/category-data/wards returns an unpaginated collection — one client request can dump the whole table. |
| 15 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/CategoryDataResource.java:50` | GET /api/military-youth/category-data/countries returns an unpaginated collection — one client request can dump the whole table. |
| 18 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/CategoryDataResource.java:55` | GET /api/military-youth/category-data/dropdown returns an unpaginated collection — one client request can dump the whole table. |
| 21 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/CitizenResource.java:36` | GET /api/military-youth/citizens/auto-complete returns an unpaginated collection — one client request can dump the whole table. |
| 24 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/CitizenResource.java:45` | GET /api/military-youth/citizens/filler-auto-complete returns an unpaginated collection — one client request can dump the whole table. |
| 36 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/DepartmentResource.java:28` | GET /api/military-youth/department/tree returns an unpaginated collection — one client request can dump the whole table. |
| 39 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/DepartmentResource.java:37` | GET /api/military-youth/department/search returns an unpaginated collection — one client request can dump the whole table. |
| 58 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/MagazineArticlesResource.java:56` | GET /api/military-youth/magazine-articles/search/all-by-Name returns an unpaginated collection — one client request can dump the whole table. |
| 78 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/MemberEvaluationResource.java:123` | GET /api/military-youth/member-evaluations/assessment-terms returns an unpaginated collection — one client request can dump the whole table. |
| 117 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/TrainingPlanResource.java:215` | GET /api/military-youth/training-plans/get-term returns an unpaginated collection — one client request can dump the whole table. |
| 130 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/TrainingResultResource.java:148` | GET /api/military-youth/training-results/get-term returns an unpaginated collection — one client request can dump the whole table. |
| 134 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/UnionEvaluationResource.java:108` | GET /api/military-youth/union-evaluations/assessment-terms returns an unpaginated collection — one client request can dump the whole table. |
| 152 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/UnionMembershipCollectionResource.java:244` | GET /api/military-youth/union-membership-collections/fee-terms returns an unpaginated collection — one client request can dump the whole table. |
| 161 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/UnionMembershipExpenditureResource.java:232` | GET /api/military-youth/union-membership-expenditures/fee-terms returns an unpaginated collection — one client request can dump the whole table. |
| 194 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthResource.java:115` | GET /api/military-youth/youths/auto-complete returns an unpaginated collection — one client request can dump the whole table. |
| 199 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthResource.java:129` | GET /api/military-youth/youths/auto-fill returns an unpaginated collection — one client request can dump the whole table. |
| 211 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthUnionResource.java:238` | GET /api/military-youth/youth-unions/tree returns an unpaginated collection — one client request can dump the whole table. |
| 215 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthUnionResource.java:258` | GET /api/military-youth/youth-unions/find-all-id-active returns an unpaginated collection — one client request can dump the whole table. |
| 224 | R3xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthUnionResource.java:367` | GET /api/military-youth/youth-unions/tree-by-user returns an unpaginated collection — one client request can dump the whole table. |
| 12 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/CategoryDataResource.java:36` | /api/military-youth/category-data/provinces returns a bare List<CategoryDataDto> — a bare JSON array leaves no room for total/page metadata. |
| 14 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/CategoryDataResource.java:42` | /api/military-youth/category-data/wards returns a bare List<CategoryDataDto> — a bare JSON array leaves no room for total/page metadata. |
| 16 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/CategoryDataResource.java:50` | /api/military-youth/category-data/countries returns a bare List<CategoryDataDto> — a bare JSON array leaves no room for total/page metadata. |
| 19 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/CategoryDataResource.java:55` | /api/military-youth/category-data/dropdown returns a bare List<CategoryDataDto> — a bare JSON array leaves no room for total/page metadata. |
| 22 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/CitizenResource.java:36` | /api/military-youth/citizens/auto-complete returns a bare List<CitizenAutoCompleteProjection> — a bare JSON array leaves no room for total/page metadata. |
| 25 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/CitizenResource.java:45` | /api/military-youth/citizens/filler-auto-complete returns a bare List<CitizenAutoCompleteProjection> — a bare JSON array leaves no room for total/page metadata. |
| 31 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/DefenseDiplomacyResource.java:170` | /api/military-youth/defense-diplomacies returns a bare List<DefenseDiplomacyDTO> — a bare JSON array leaves no room for total/page metadata. |
| 37 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/DepartmentResource.java:28` | /api/military-youth/department/tree returns a bare List<DepartmentTreeResponse> — a bare JSON array leaves no room for total/page metadata. |
| 40 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/DepartmentResource.java:37` | /api/military-youth/department/search returns a bare List<DepartmentTreeResponse> — a bare JSON array leaves no room for total/page metadata. |
| 59 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/MagazineArticlesResource.java:56` | /api/military-youth/magazine-articles/search/all-by-Name returns a bare List<MagazineArticles> — a bare JSON array leaves no room for total/page metadata. |
| 64 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/MagazineArticlesResource.java:158` | /api/military-youth/magazine-articles returns a bare List<MagazineArticlesDTO> — a bare JSON array leaves no room for total/page metadata. |
| 71 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/MagazineListQuarterlyResource.java:173` | /api/military-youth/magazine-list-quarterlies returns a bare List<MagazineListQuarterlyDTO> — a bare JSON array leaves no room for total/page metadata. |
| 79 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/MemberEvaluationResource.java:123` | /api/military-youth/member-evaluations/assessment-terms returns a bare List<Integer> — a bare JSON array leaves no room for total/page metadata. |
| 113 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/TrainingPlanResource.java:171` | /api/military-youth/training-plans returns a bare List<TrainingPlanDTO> — a bare JSON array leaves no room for total/page metadata. |
| 118 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/TrainingPlanResource.java:215` | /api/military-youth/training-plans/get-term returns a bare List<Integer> — a bare JSON array leaves no room for total/page metadata. |
| 131 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/TrainingResultResource.java:148` | /api/military-youth/training-results/get-term returns a bare List<Integer> — a bare JSON array leaves no room for total/page metadata. |
| 135 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/UnionEvaluationResource.java:108` | /api/military-youth/union-evaluations/assessment-terms returns a bare List<Integer> — a bare JSON array leaves no room for total/page metadata. |
| 153 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/UnionMembershipCollectionResource.java:244` | /api/military-youth/union-membership-collections/fee-terms returns a bare List<String> — a bare JSON array leaves no room for total/page metadata. |
| 162 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/UnionMembershipExpenditureResource.java:232` | /api/military-youth/union-membership-expenditures/fee-terms returns a bare List<String> — a bare JSON array leaves no room for total/page metadata. |
| 195 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/YouthResource.java:115` | /api/military-youth/youths/auto-complete returns a bare List<YouthAutocompleteResponse> — a bare JSON array leaves no room for total/page metadata. |
| 200 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/YouthResource.java:129` | /api/military-youth/youths/auto-fill returns a bare List<YouthAutocompleteResponse> — a bare JSON array leaves no room for total/page metadata. |
| 212 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/YouthUnionResource.java:238` | /api/military-youth/youth-unions/tree returns a bare List<YouthUnionTreeResponse> — a bare JSON array leaves no room for total/page metadata. |
| 216 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/YouthUnionResource.java:258` | /api/military-youth/youth-unions/find-all-id-active returns a bare List<YouthUnion> — a bare JSON array leaves no room for total/page metadata. |
| 225 | R3xx-02 | INFO | `src/main/java/com/viettel/tcct/web/rest/YouthUnionResource.java:367` | /api/military-youth/youth-unions/tree-by-user returns a bare List<YouthUnionTreeResponse> — a bare JSON array leaves no room for total/page metadata. |
| 51 | R1xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/InnovationByUnitResource.java:79` | Collection path segment "innovation-by-unit" is singular — collections use plural nouns so one concept keeps one URL (AIP-131, AIP-122). |
| 53 | R1xx-01 | WARN | `src/main/java/com/viettel/tcct/web/rest/InnovationByUnitResource.java:99` | Collection path segment "innovation-by-unit" is singular — collections use plural nouns so one concept keeps one URL (AIP-131, AIP-122). |
| 121 | R2xx-05 | INFO | `src/main/java/com/viettel/tcct/web/rest/TrainingPlanResource.java:240` | Endpoint approve (PUT /api/military-youth/training-plans/approve) looks like an action (non-CRUD) — AIP-136 custom methods read POST <collection>/{id}:<action> (verify manually). |
| 125 | R2xx-05 | INFO | `src/main/java/com/viettel/tcct/web/rest/TrainingResultResource.java:100` | Endpoint approve (PUT /api/military-youth/training-results/approve) looks like an action (non-CRUD) — AIP-136 custom methods read POST <collection>/{id}:<action> (verify manually). |
| 165 | R2xx-05 | INFO | `src/main/java/com/viettel/tcct/web/rest/VauthzController.java:44` | Endpoint syncDepartment (GET /api/military-youth/auth/sync-departments) looks like an action (non-CRUD) — AIP-136 custom methods read POST <collection>/{id}:<action> (verify manually). |
| 184 | R2xx-05 | INFO | `src/main/java/com/viettel/tcct/web/rest/YouthMovementGuidelinesResource.java:193` | Endpoint updateStatus (PUT /api/military-youth/youth-movement-guidelines/update-status-uncheck/{id}) looks like an action (non-CRUD) — AIP-136 custom methods read POST <collection>/{id}:<action> (verify manually). |
| 43 | R2xx-02 | WARN | `src/main/java/com/viettel/tcct/web/rest/DocumentResource.java:49` | POST endpoint create looks like a resource create (single DocumentDetailResponse response, no action path) but does not declare 201 Created — verify that this POST creates, and answer 201. |
| 47 | R2xx-02 | WARN | `src/main/java/com/viettel/tcct/web/rest/InnovationByUnitResource.java:51` | POST endpoint createInnovationByUnit looks like a resource create (single ApiResponse<Void> response, no action path) but does not declare 201 Created — verify that this POST creates, and answer 201. |
| 80 | R2xx-02 | WARN | `src/main/java/com/viettel/tcct/web/rest/MemberEvaluationResource.java:139` | POST endpoint saveMemberEvaluation looks like a resource create (single MemberEvaluation response, no action path) but does not declare 201 Created — verify that this POST creates, and answer 201. |
| 92 | R2xx-02 | WARN | `src/main/java/com/viettel/tcct/web/rest/MovementReportResource.java:37` | POST endpoint saveDraft looks like a resource create (single ApiResponse<MovementReportRegistrationResponse> response, no action path) but does not declare 201 Created — verify that this POST creates, and answer 201. |
| 94 | R2xx-02 | WARN | `src/main/java/com/viettel/tcct/web/rest/MovementReportResource.java:48` | POST endpoint register looks like a resource create (single ApiResponse<MovementReportRegistrationResponse> response, no action path) but does not declare 201 Created — verify that this POST creates, and answer 201. |
| 95 | R2xx-02 | WARN | `src/main/java/com/viettel/tcct/web/rest/MovementReportResource.java:93` | POST endpoint saveResultDraft looks like a resource create (single ApiResponse<MovementReportRegistrationResponse> response, no action path) but does not declare 201 Created — verify that this POST creates, and answer 201. |
| 100 | R2xx-02 | WARN | `src/main/java/com/viettel/tcct/web/rest/ReportResource.java:53` | POST endpoint create looks like a resource create (single ReportDTO response, no action path) but does not declare 201 Created — verify that this POST creates, and answer 201. |
| 136 | R2xx-02 | WARN | `src/main/java/com/viettel/tcct/web/rest/UnionEvaluationResource.java:116` | POST endpoint createUnionEvaluation looks like a resource create (single UnionEvaluation response, no action path) but does not declare 201 Created — verify that this POST creates, and answer 201. |
| 145 | R2xx-02 | WARN | `src/main/java/com/viettel/tcct/web/rest/UnionMembershipCollectionResource.java:71` | POST endpoint create looks like a resource create (single UnionMembershipCollectionDTO response, no action path) but does not declare 201 Created — verify that this POST creates, and answer 201. |
| 155 | R2xx-02 | WARN | `src/main/java/com/viettel/tcct/web/rest/UnionMembershipExpenditureResource.java:66` | POST endpoint create looks like a resource create (single UnionMembershipExpenditureDTO response, no action path) but does not declare 201 Created — verify that this POST creates, and answer 201. |
| 166 | R2xx-02 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthInnovationRegistrationResource.java:73` | POST endpoint create looks like a resource create (single YouthInnovationRegistrationDTO response, no action path) but does not declare 201 Created — verify that this POST creates, and answer 201. |
| 185 | R2xx-02 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthResource.java:58` | POST endpoint createYouth looks like a resource create (single YouthDTO response, no action path) but does not declare 201 Created — verify that this POST creates, and answer 201. |
| 44 | R5xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/DocumentResource.java:49` | POST /api/military-youth/documents relies on the implicit 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success. |
| 48 | R5xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/InnovationByUnitResource.java:51` | POST /api/military-youth/innovation-by-unit relies on the implicit 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success. |
| 81 | R5xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/MemberEvaluationResource.java:139` | POST /api/military-youth/member-evaluations relies on the implicit 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success. |
| 93 | R5xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/MovementReportResource.java:37` | POST /api/military-youth/movement-reports/draft relies on the implicit 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success. |
| 96 | R5xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/MovementReportResource.java:93` | POST /api/military-youth/movement-reports/result/draft relies on the implicit 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success. |
| 101 | R5xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/ReportResource.java:53` | POST /api/military-youth/reports relies on the implicit 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success. |
| 107 | R5xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/SystemCodeConfigResource.java:38` | POST /api/military-youth/system-code-configs/generate-code relies on the implicit 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success. |
| 138 | R5xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/UnionEvaluationResource.java:116` | POST /api/military-youth/union-evaluations relies on the implicit 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success. |
| 146 | R5xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/UnionMembershipCollectionResource.java:71` | POST /api/military-youth/union-membership-collections relies on the implicit 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success. |
| 156 | R5xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/UnionMembershipExpenditureResource.java:66` | POST /api/military-youth/union-membership-expenditures relies on the implicit 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success. |
| 167 | R5xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthInnovationRegistrationResource.java:73` | POST /api/military-youth/youth-innovation-registrations relies on the implicit 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success. |
| 186 | R5xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthResource.java:58` | POST /api/military-youth/youths relies on the implicit 200 while creating a resource — declare 201 Created so clients can tell creation from a plain success. |
| 33 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/DefenseDiplomacyResource.java:217` | Path segment "delete" states the CRUD verb "delete" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 74 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/MagazineListQuarterlyResource.java:227` | Path segment "delete" states the CRUD verb "delete" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 97 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/MovementReportResource.java:135` | Path segment "update" states the CRUD verb "update" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 99 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/MovementReportResource.java:146` | Path segment "update" states the CRUD verb "update" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 104 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/ReportResource.java:161` | Path segment "delete" states the CRUD verb "delete" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 115 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/TrainingPlanResource.java:215` | Path segment "get-term" states the CRUD verb "get" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 119 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/TrainingPlanResource.java:221` | Path segment "delete" states the CRUD verb "delete" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 128 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/TrainingResultResource.java:148` | Path segment "get-term" states the CRUD verb "get" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 149 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/UnionMembershipCollectionResource.java:211` | Path segment "delete" states the CRUD verb "delete" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 159 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/UnionMembershipExpenditureResource.java:206` | Path segment "delete" states the CRUD verb "delete" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 170 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/YouthInnovationRegistrationResource.java:226` | Path segment "delete" states the CRUD verb "delete" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 182 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/YouthMovementGuidelinesResource.java:193` | Path segment "update-status-uncheck" states the CRUD verb "update" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 213 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/YouthUnionResource.java:258` | Path segment "find-all-id-active" states the CRUD verb "find" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 226 | R1xx-02 | ERROR | `src/main/java/com/viettel/tcct/web/rest/YouthUnionResource.java:382` | Path segment "find-by-id" states the CRUD verb "find" — the HTTP verb already carries the action (AIP-131, AIP-133, AIP-135). |
| 34 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/DefenseDiplomacyResource.java:217` | DELETE endpoint delete declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 46 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/DocumentResource.java:74` | DELETE endpoint deleteHard declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 54 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/InnovationByUnitResource.java:106` | DELETE endpoint delete declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 75 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/MagazineListQuarterlyResource.java:227` | DELETE endpoint delete declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 76 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/MemberEvaluationResource.java:79` | DELETE endpoint delete declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 105 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/ReportResource.java:161` | DELETE endpoint delete declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 120 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/TrainingPlanResource.java:221` | DELETE endpoint delete declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 132 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/UnionEvaluationResource.java:79` | DELETE endpoint deleteMultiple declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 150 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/UnionMembershipCollectionResource.java:211` | DELETE endpoint delete declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 160 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/UnionMembershipExpenditureResource.java:206` | DELETE endpoint delete declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 171 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthInnovationRegistrationResource.java:226` | DELETE endpoint delete declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 177 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthMovementGuidelinesResource.java:116` | DELETE endpoint deleteMultiple declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 188 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthResource.java:79` | DELETE endpoint deleteYouth declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 204 | R2xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthUnionResource.java:197` | DELETE endpoint deleteYouthUnion declares a request body (List<String>) — model bulk deletes as query parameters or a batch path instead. |
| 187 | R2xx-06 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthResource.java:65` | PUT endpoint partialUpdateYouth takes payload YouthUpdateRequest but returns YouthDTO — the mismatch looks like a partial update, which belongs to PATCH. |
| 56 | R1xx-04 | WARN | `src/main/java/com/viettel/tcct/web/rest/MagazineArticlesResource.java:56` | Path segment "all-by-Name" is not kebab-case — keep one segment convention (AIP-122). |
| 60 | R4xx-01 | ERROR | `src/main/java/com/viettel/tcct/web/rest/MagazineArticlesResource.java:56` | GET /api/military-youth/magazine-articles/search/all-by-Name exposes the ORM entity MagazineArticles — the database schema and lazy relations leak into the wire contract. |
| 137 | R4xx-01 | ERROR | `src/main/java/com/viettel/tcct/web/rest/UnionEvaluationResource.java:116` | POST /api/military-youth/union-evaluations exposes the ORM entity UnionEvaluation — the database schema and lazy relations leak into the wire contract. |
| 142 | R4xx-01 | ERROR | `src/main/java/com/viettel/tcct/web/rest/UnionEvaluationResource.java:150` | PUT /api/military-youth/union-evaluations/{id} exposes the ORM entity UnionEvaluation — the database schema and lazy relations leak into the wire contract. |
| 151 | R4xx-01 | ERROR | `src/main/java/com/viettel/tcct/web/rest/UnionMembershipCollectionResource.java:231` | POST /api/military-youth/union-membership-collections/import exposes the ORM entity UnionMembershipCollection — the database schema and lazy relations leak into the wire contract. |
| 172 | R4xx-01 | ERROR | `src/main/java/com/viettel/tcct/web/rest/YouthInnovationRegistrationResource.java:285` | POST /api/military-youth/youth-innovation-registrations/import exposes the ORM entity InnovationProject — the database schema and lazy relations leak into the wire contract. |
| 189 | R4xx-01 | ERROR | `src/main/java/com/viettel/tcct/web/rest/YouthResource.java:86` | POST /api/military-youth/youths/import exposes the ORM entity Youth — the database schema and lazy relations leak into the wire contract. |
| 205 | R4xx-01 | ERROR | `src/main/java/com/viettel/tcct/web/rest/YouthUnionResource.java:205` | POST /api/military-youth/youth-unions/import exposes the ORM entity YouthUnion — the database schema and lazy relations leak into the wire contract. |
| 217 | R4xx-01 | ERROR | `src/main/java/com/viettel/tcct/web/rest/YouthUnionResource.java:258` | GET /api/military-youth/youth-unions/find-all-id-active exposes the ORM entity YouthUnion — the database schema and lazy relations leak into the wire contract. |
| 183 | R1xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthMovementGuidelinesResource.java:193` | Variable {id} sits under the action segment "update-status-uncheck" — serve the item under its resource: /<collection>/{id} (AIP-127). |
| 227 | R1xx-03 | WARN | `src/main/java/com/viettel/tcct/web/rest/YouthUnionResource.java:382` | Variable {id} sits under the action segment "find-by-id" — serve the item under its resource: /<collection>/{id} (AIP-127). |
| 197 | R2xx-01 | ERROR | `src/main/java/com/viettel/tcct/web/rest/YouthResource.java:116` | GET endpoint autoComplete declares a request body (YouthAutocompleteRequest) — many HTTP clients silently drop GET bodies. |
| 229 | R6xx-02 | INFO | `src/main/proto/entity/defense_diplomacy.proto:17` | rpc PartialUpdateDefenseDiplomacy in service DefenseDiplomacyService is not a Standard Method name (Get*/List*/Create*/Update*/Delete*) or a VerbNoun custom method — the name leaks into every generated client. |
| 230 | R6xx-02 | INFO | `src/main/proto/entity/defense_participants.proto:16` | rpc PartialUpdateDefenseParticipants in service DefenseParticipantsService is not a Standard Method name (Get*/List*/Create*/Update*/Delete*) or a VerbNoun custom method — the name leaks into every generated client. |
| 231 | R6xx-02 | INFO | `src/main/proto/entity/defense_schedule.proto:17` | rpc PartialUpdateDefenseSchedule in service DefenseScheduleService is not a Standard Method name (Get*/List*/Create*/Update*/Delete*) or a VerbNoun custom method — the name leaks into every generated client. |

## Phụ lục C — Truy vết từng con số

| Con số trong tài liệu | Giá trị | Nguồn (artifact + lệnh) |
|---|---|---|
| File target / scanned / skip | 774 / 753 / 9 | `scan.json` `.summary`; fingerprint A.1 (`find … sha256sum` trong sandbox, w5-02 §4.1) |
| Findings theo severity | 244 = 24E/143W/77I | `jq '.summary.findings' scan.json` |
| 18 rule bắn / bảng per-rule | 18 | `jq group_by(.ruleId)` × `scan.json` |
| 182 endpoint / 184 handler / 24 controller | 182 / 184 / 24 | `compare.json` (`refTotalUnique`, `refTotalList`, `refMeta`) |
| Recall 100% (182/182, FN 0, thừa 0) | 1.0 | `compare.json` (`matched`, `fn`, `extra`, `recallExact`) |
| Phân bố verb GET 87 / POST 54 / DELETE 20 / PUT 19 / PATCH 4 | entry-level | `compare.json` `perClass` + `swagger-ref.json` |
| 129 TP / 115 FP / FP share 47.1% | 129/115 | tái-đếm script fail-closed 2026-10-09 (31/31 PASS) × `scan.json`; đối chiếu `audit-results.json` `.military_youth.counts` |
| ERROR 23/24 thật (95.8%) | 23 TP / 1 FP | bảng triage × severity trong `scan.json` |
| WARN 64/79 · INFO 42/35 | 64 TP/79 FP · 42 TP/35 FP | như trên — **lưu ý**: dòng prose w5-04 §2 ghi 75/68 + 31/46 là lệch với bảng per-rule + machine sheet của chính w5-04; tài liệu này dùng số script (đã đối chiếu cả 3 nguồn máy) |
| 6 FN nghi vấn | 6 | `audit-results.json` `.military_youth.fn_candidates` |
| Hotspot TP theo file | YouthUnion 12, Youth 10, … | tái-đếm từ Phụ lục B × `scan.json` |
| 13 endpoint create-200 | 13 unique span | hợp R2xx-02 ∪ R5xx-03 (TP) theo location trong `scan.json` |
| 7 span trùng R1xx-02∩R2xx-04 · 20 span trùng R3xx-01∩R3xx-02 | 7 / 20 | hợp/giao location TP trong `scan.json` |
| 0 suppression | 0 | `scan.json` `.summary.suppressed` + w5-02 §1.2 (0 file `.vanguard*` trong target) |

Lệnh tái-đếm máy (fail-closed, 31/31 PASS, sandbox dind-sandbox:29 2026-10-09): `reports/w5-05-verify.py`
(chạy kèm `w505-triage-map.py` — bản path-patched của `reports/w5-04-triage-map.py` @ branch
`agent-w5-04`); input = 2 artifact sha256-pinned ở A.5.
