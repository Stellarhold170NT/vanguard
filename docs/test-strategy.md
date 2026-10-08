# Vanguard — Test Strategy & Corpus Plan (w1-05)

**Role**: architect · **Ngày**: 2026-10-08 · **Trạng thái**: v1 — chốt phương pháp chứng minh chất lượng cho charter §7
**Đọc cho ai**: w2-07 (tầng 1), w4-01 (tầng 2 + §3), w4-02 (tầng 3 + §4), w4-03 (tầng 4 + §5), w4-04/w4-05 (§7 + bảng §2), W5 (§6 + §7).
**Nguồn ràng buộc**: `docs/charter.md` §3 (taxonomy 26 rule), §5.1 (repo layout: `testdata/`), §6.3 (exit codes), §6.8 (SARIF), §7 (đích R7), §12 (hợp đồng W4/W5).

---

## Mục lục

- [0. Cách đọc document này](#0-cách-đọc-document-này)
- [1. Nguyên tắc chất lượng](#1-nguyên-tắc-chất-lượng)
- [2. Kiến trúc test 5 tầng](#2-kiến-trúc-test-5-tầng)
- [3. Corpus plan — adversarial (w4-01)](#3-corpus-plan--adversarial-w4-01)
- [4. Mutation plan (w4-02)](#4-mutation-plan-w4-02)
- [5. Perf & determinism protocol (w4-03)](#5-perf--determinism-protocol-w4-03)
- [6. Real-fire criteria (W5)](#6-real-fire-criteria-w5)
- [7. Giao thức FP/FN audit](#7-giao-thức-fpfn-audit)
- [8. Hộp công cụ đo](#8-hộp-công-cụ-đo)
- [9. Quy tắc anti-gaming](#9-quy-tắc-anti-gaming)
- [10. Điều kiện sửa plan này](#10-điều-kiện-sửa-plan-này)

---

## 0. Cách đọc document này

Mỗi task chất lượng đọc **đúng mục của mình** và làm theo — không tự thiết kế lại phương pháp (acceptance
của task w1-05: "mỗi task W4 có thể đọc plan này và làm ngay"). Nếu thực tế đo được khác đích số ở đây →
ghi nguyên bản vào report của task đó và đề xuất điều chỉnh theo §10; **không** tự sửa ngưỡng im lặng.

| Task | Đọc mục | Phải xuất ra |
|---|---|---|
| w2-07 | §2.1 | `internal/golden` harness + snapshot; 12 case gợi ý; verdict CH2 |
| w4-01 | §3 toàn bộ | `testdata/adversarial/` 161 case per-rule + 10 global; `manifest.yaml` |
| w4-02 | §4 toàn bộ | `internal/mutation` + mutator bộ MUST; `testdata/mutation-results.json`; blind-spot report |
| w4-03 | §5 toàn bộ | `tools/gen-bench-repo/`; số đo perf; idempotency 6 cặp; `testdata/perf-results.json` |
| w4-04 | §7 | sample set + labeling sheet + audit harness; materialize `docs/fpfn-protocol.md` |
| w4-05 | §2 + §7 (phần tổng hợp) | quality report GATE CH3 đối chiếu ratchet level 2 |
| w5-01..05 | §6 | SOP, artifacts scan, recall script, triage sheet, baseline |
| w5-06 | §7 (tổng hợp) + §2 | postmortem + verdict CH4 |

---

## 1. Nguyên tắc chất lượng

1. **FP là ưu tiên số 1** (charter §7): mỗi tầng test đều phải đo được cả hai chiều — bắt được vi phạm
   (recall/catch) và không báo oan (precision/FP). Corpus luôn có cặp `vio` (đo FN) và `ok` (đo FP).
2. **Trung thực tuyệt đối về số liệu**: expected-miss ghi vào manifest **trước khi chạy**; không "tinh
   chỉnh kỳ vọng" sau khi thấy kết quả xấu; không viết case để làm đẹp catch-rate (protocol w4-01/w4-02).
   Mỗi số trong mọi report truy vết được về artifact (JSON/log/lệnh tái chạy).
3. **Mọi ngưỡng phải đo được bằng công cụ có sẵn** — sandbox shell (`time`, `find`, `wc`, `sha256sum`,
   `cmp`), JSON output của chính vanguard. §8 liệt kê lệnh cụ thể cho từng metric; plan này **không**
   đề xuất metric nào không có cách đo.
4. **Không copy code military-youth** vào testdata (bản quyền/nhạy cảm — charter §12): corpus chỉ mô
   phỏng pattern với domain trung tính (library/training/inventory), tên biến/file tự đặt.
5. **Mỗi tầng có gate số + artifact**, không tầng nào "pass theo cảm giác". Bảng §2 là hợp đồng.
6. **Tầng dưới rẻ chạy trước**: golden (giây) → corpus (phút) → mutation (≤5 phút) → perf (chục phút) →
   real-fire (một lần mỗi release). Lỗi phát hiện ở tầng rẻ không được để trôi xuống tầng đắt.

---

## 2. Kiến trúc test 5 tầng

| # | Tầng | Task | Đo cái gì | Gate số | Artifact |
|---|---|---|---|---|---|
| 1 | Fixture golden | w2-07 | Engine + render + config + suppression hoạt động **đúng như snapshot** | 100% snapshot match; `go test ./...` green; 2 lần chạy liên tiếp pass | `testdata/golden/`, `internal/golden` |
| 2 | Adversarial corpus | w4-01 | Rule **phân biệt** ok/vio trên code khó; đo FP-side từ nay | manifest-agreement ≥ 85% (baseline, expected-miss đã khai) | `testdata/adversarial/` + `manifest.yaml` |
| 3 | Mutation harness | w4-02 | Độ phủ **thực** của rule: code đúng biến sai tự động phải bị bắt | catch-rate tổng ≥ 90%; per-rule (có case) không 0%; 2 lần chạy cùng số | `internal/mutation`, `testdata/mutation-results.json` |
| 4 | Perf & determinism | w4-03 | Tốc độ + ổn định tuyệt đối của output | p95 ≤ 60s / 10k file; byte-identical 6 cặp so sánh; 0 crash edge | `tools/gen-bench-repo/`, `testdata/perf-results.json` |
| 5 | Real-fire | w5-01..05 | Giá trị thật trên military-youth | recall discovery ≥ 95%; 100% finding có verdict | `artifacts/w5-02/`, triage sheets, baseline |

Tổng hợp gate: **CH2** (w2-07, auto-green) → **CH3** (w4-05: ratchet level 2 = FP < 15% ∧ catch-rate ≥ 90%
∧ 0 critical robustness fail) → **CH4** (w5-06 postmortem).

### 2.1. Tầng 1 — Fixture golden (w2-07)

**Mục tiêu**: chứng minh engine/config/suppression/render hoạt động đúng từng chi tiết, và mọi thay đổi
hành vi output là **chủ ý + review được**.

**Cấu trúc**:

- Harness: `internal/golden` (Go test helper) — chạy binary qua `exec` (đúng như CI dùng) trên thư mục
  fixture `testdata/golden/<case-id>/`, snapshot 3 format `snapshot.pretty` / `snapshot.json` /
  `snapshot.sarif` cạnh fixture, so sánh sau **path normalization** (uri tương đối từ scan root — bẫy
  đã ghi ở W2).
- Lệnh `make golden-update` để cập nhật snapshot có chủ ý; diff của snapshot phải xuất hiện trong review.

**Tiêu chí pass (đầy đủ)**:

- [ ] `go test ./internal/golden/...` green; `go test ./...` gồm ≥ 1 integration case chạy binary qua `exec`.
- [ ] 2 lần chạy liên tiếp pass — không flaky (nếu flaky → bug determinism, quay về §5.4).
- [ ] Set fixture W2 ≥ 10 case trải đủ 6 diện engine; danh sách gợi ý (w2-07 tự đặt tên cuối):
      1. `default-scan` — không config, scan stub, findings đúng mock §6.6–6.8.
      2. `config-discovery` — `.vanguard.yaml` ở thư mục cha được phát hiện leo lên.
      3. `config-explicit` — `--config` thắng discovery.
      4. `suppression-inline` — `vanguard:ignore` dòng đơn + hint hiển thị 1 lần/nhóm.
      5. `suppression-block` — khối ignore nhiều dòng.
      6. `severity-display-vs-exit` — `--severity` đổi hiển thị, KHÔNG đổi exit code (§6.3).
      7. `exit-codes` — 3 case con: chỉ WARN (exit 0) / có ERROR (exit 1) / config sai schema (exit 2 + stderr).
      8. `no-api-surface` — repo không có java → exit 0 + summary thông báo rõ.
      9. `multi-format-consistency` — cùng 1 scan, 3 format cùng bộ finding (so chéo).
      10. `rule-disable-config` — tắt rule qua config → finding biến mất.
      11. `sort-stability` — nhiều findings trùng (file,line) sort ổn định theo §6.6.
      12. `data-driven-proof` — thêm 1 rule mới bằng **chỉ data** (metadata YAML), không sửa harness → test vẫn green.
- [ ] Snapshot nằm trong repo, review được bằng diff; không nhạy đường dẫn tuyệt đối, timezone, hostname.

**Chống-gaming**: golden green không chứng minh rule đúng ngữ nghĩa — chỉ chứng minh output ổn định và
khớp kỳ vọng đã review. Ngữ nghĩa do tầng 2/3/5 chứng minh.

### 2.2. Tầng 2 — Adversarial corpus (w4-01)

**Mục tiêu**: code "gần đúng mà sai" và "nhìn nghi mà đúng" phải được phân loại đúng. Chi tiết phân bổ,
quy ước, near-miss → **§3** (là hợp đồng bắt buộc cho w4-01).

**Tiêu chí pass**:

- [ ] Đủ 161 case per-rule + 10 case global (§3.3, §3.5), mỗi file ≤ 40 dòng, header comment đúng §3.1.
- [ ] Mỗi rule ≥ 3 `vio` + ≥ 2 `ok`; mỗi rule có ≥ 1 cặp near-miss mỗi loại (§3.2).
- [ ] `vanguard scan` trên corpus không crash; exit code đúng quy tắc §6.3 (corpus chứa vio ERROR →
      exit 1 là **hành vi đúng**, không phải lỗi tool; chỉ exit 2 mới là lỗi).
- [ ] Manifest-agreement ≥ 85% tại thời điểm w4-01: số case mà kết quả thực khớp `expect` trong manifest
      / tổng case. Mismatch phải được phân loại: `expected-miss` (đã khai trước trong manifest kèm lý do
      kỹ thuật) hoặc `surprise` (ghi vào report — dữ liệu postmortem w5-06). Baseline thấp là bình thường —
      điểm xuất phát, w4-02 sẽ đo lại sau mutation.

### 2.3. Tầng 3 — Mutation harness (w4-02)

**Mục tiêu**: đo rule bắt được bao nhiêu % vi phạm được **sinh tự động** — khử yếu tố "case được viết
khéo theo rule". Danh mục mutator, ngưỡng, cách đo blind-spot → **§4** (hợp đồng bắt buộc cho w4-02).

**Tiêu chí pass**:

- [ ] Đủ bộ mutator MUST (§4.1 — 17 mutator); mỗi rule in-scope có ≥ 1 mutator hoặc được đánh dấu
      `no-mutator` kèm lý do (chỉ chấp nhận cho rule INFO optional, §4.1).
- [ ] Catch-rate tổng ≥ 90%; per-rule: rule nào có ≥ 1 mutation case thì catch > 0%.
- [ ] Harness idempotent + deterministic: 2 lần chạy = cùng bộ số trong JSON.
- [ ] `make mutation` ≤ 5 phút trong CI (công thức chọn case §4.4).
- [ ] Case bị mutator phá cú pháp → **loại khỏi mẫu số** (không tính miss) nhưng phải được log đếm trong JSON.
- [ ] Blind-spot report: mọi ô (rule, mutator) miss được liệt kê kèm fixture đã mutate + giả thuyết
      `parser-gap` / `predicate-gap` / `config-off` — input cho postmortem w5-06.

### 2.4. Tầng 4 — Perf & determinism (w4-03)

**Mục tiêu**: vanguard chạy nhanh trên repo lớn và output **byte-identical** giữa các lần chạy — điều kiện
để làm CI gate. Protocol đo chi tiết → **§5** (hợp đồng bắt buộc cho w4-03).

**Tiêu chí pass**:

- [ ] p95 ≤ 60s cho full scan bench repo ≥ 10.000 file java (5 runs, sandbox 2cpu/4Gi — công thức §5.2/§5.3).
- [ ] Idempotency: 2 lần chạy × 2 nguồn (corpus + bench repo) × 3 format = 6 cặp so sánh sha256 identical.
- [ ] Edge cases §5.5: không crash, exit code đúng định nghĩa, số finding hữu hạn.
- [ ] Kết quả JSON `testdata/perf-results.json` + môi trường đo ghi rõ trong `reports/w4-03.md`.

### 2.5. Tầng 5 — Real-fire (W5)

**Mục tiêu**: chứng minh giá trị trên repo thật military-youth. Tiêu chí chi tiết → **§6**.

**Tiêu chí pass (tổng)**:

- [ ] Recall discovery ≥ 95%: endpoint mà swagger/springdoc liệt kê được phát hiện qua scan source
      (đếm bằng script vì 185 endpoint > 20 — §6.2), đối chiếu qua steel khi runtime chạy được.
- [ ] 100% finding của scan thật có verdict TP/FP/FN theo §7 — không finding nào bỏ trống.
- [ ] Artifacts đầy đủ, truy vết được (w5-02), baseline cho team (w5-05), postmortem có danh sách
      adjustment đo được (w5-06).
- [ ] Nếu recall < 95%: **không phải thất bại quy trình** — liệt kê từng endpoint miss + giả thuyết
      nguyên nhân, đưa vào postmortem (protocol w5-03).

---

## 3. Corpus plan — adversarial (w4-01)

### 3.1. Quy ước testdata (bắt buộc theo nguyên văn)

```
testdata/adversarial/
├── manifest.yaml                    # nguồn sự thật — w4-02 harness đọc file này
├── R1xx-01/
│   ├── ok-plural-standard.java      # ok-<slug>: code ĐÚNG — rule KHÔNG được hit
│   ├── vio-singular-path.java       # vio-<slug>: code SAI — rule PHẢI hit ≥ 1
│   └── ...
├── R6xx-02/
│   └── ...
└── _global/                         # corner case toàn cục — §3.5 (underscore để sort đầu, không phải rule-id)
    └── ...
```

- **Thư mục** = rule ID nguyên văn `R<F>xx-<NN>` (đúng §3 charter, dễ grep). **Không** dùng slug làm tên thư mục.
- **Tên file**: `{ok,vio}-<slug>.java`; slug kebab-case tiếng Anh ≤ 5 từ, mô tả **điểm lừa** chứ không mô tả rule
  (`vio-update-draft-path` ✓, `vio-test1` ✗). Slug duy nhất trong thư mục rule.
- **Header comment 2 dòng đầu mỗi file** (dòng 1 máy đọc, dòng 2 người đọc):

```java
// corpus: R1xx-02 vio update-draft-path
// lure: POST /update/draft — động từ "update" ngụy trang dưới dạng action-phrase hai tầng.
```

- **Kích thước**: mỗi case ≤ 40 dòng (bẫy w4-01 — case khổng lồ khó debug). Đủ annotation/type để
  tree-sitter parse được; **không yêu cầu compile** (không gate bằng javac).
- **Syntactically valid Java** bắt buộc (nguồn kiểm chứng: chính vanguard `--verbose` parse diagnostics
  ở w3-01; case parse-lỗi bị loại khỏi corpus, không được dùng làm "case khó").
- **Mutation-ready** (áp cho ok-case): ok-case của mỗi rule phải chứa cấu trúc mà mutator chính của rule
  đó (§3.3, cột "mutator") có thể biến thành vi phạm — vì w4-02 sinh mutation từ ok-case corpus.
- **Domain trung tính**: library / training / inventory; tên class tự đặt; cấm copy nguyên xi code
  military-youth và cấm tên org (R8 — cả annotation org-specific như `@VauthzCheck` không xuất hiện).

### 3.2. Định nghĩa near-miss

> **Near-miss** = case mà khoảng cách giữa code và đặc trưng kích hoạt (trigger signature) của rule chỉ
> còn **một dấu hiệu duy nhất** — one-token distance: 1 ký tự casing, 1 hậu tố tên, 1 annotation, 1 kiểu
> trả về, hoặc 1 segment path.

Hai chiều tấn công, mỗi rule phải có ≥ 1 cặp mỗi loại:

| Chiều | Prefix | Nhiệm vụ | Đo | Ví dụ (R1xx-02) |
|---|---|---|---|---|
| FN-side | `vio-` | vi phạm **thật** nhưng ngụy trang để né heuristic | rule miss = FN | `@DeleteMapping("/del-records")` — động từ viết tắt `del`; `@PostMapping("/statistics/rebuild")` — `rebuild` là động từ sau danh từ |
| FP-side | `ok-` | code **đúng** nhưng surface trùng trigger | rule hit = FP | `@GetMapping("/budget-getter/{id}")` — `getter` là danh từ nghiệp vụ; `@PostMapping("/updates")` — `updates` là danh từ số nhiều (bản cập nhật) |

Case near-miss được đánh dấu `tier: near-miss` trong manifest; case thường `tier: baseline`. Tỷ lệ khuyến
nghi: ~50% case mỗi rule là near-miss (số vio/ok trong §3.3 đã tính sẵn phần này).

### 3.3. Bảng phân bổ 161 case per-rule (≥ 150 yêu cầu)

Ươm mầm: mỗi rule 3 vio + 2 ok (= 130); phần dư **31** ưu tiên họ có nhiều FP tiềm năng (§3.4).
Cột "mutator chính" = mutator MUST §4.1 thao tác trên ok-case của rule (điều kiện mutation-ready §3.1).

| Rule | Slug | Sev | vio | ok | Tổng | Mutator chính | Điểm lừa trọng tâm (ví dụ case) |
|---|---|---|---|---|---|---|---|
| R1xx-01 | plural-collection | WARN | 4 | 3 | 7 | `singularize-path` | singular `/book`; danh từ không đếm được `/equipment` là **ok**; camel plural `/bookShelves` |
| R1xx-02 | no-verb-path | ERROR | 5 | 3 | 8 | `inject-verb-path` | `update/draft`; `del-records` viết tắt; noun chứa verb `getter`, `updates`, `preview` |
| R1xx-03 | resource-path-pattern | WARN | 4 | 3 | 7 | `flatten-resource-path` | `/find-by-id/{id}` phẳng; `/a/{id}/b/{subId}` đúng là ok; path 3 tầng lồng |
| R1xx-04 | path-casing | WARN | 4 | 3 | 7 | `mis-case-path` | `all-by-Name`; acronym `/api-key` vs `/apiKey`; biến `/{YouthId}` hoa |
| R1xx-05 | id-field-naming | INFO | 3 | 3 | 6 | `mix-id-field` ◇ | trộn `id` + `youId`; `userId` nhất quán là ok; key JSON trộn |
| R2xx-01 | get-no-body | ERROR | 4 | 3 | 7 | `body-into-get` | `@RequestBody` trong GET; `@RequestPart` (gần nhầm); GET + query object là **ok** |
| R2xx-02 | post-creates-201 | WARN | 3 | 3 | 6 | `remove-status-mapping` | POST trả 200; POST trả 201 qua `@ResponseStatus` là ok; POST trả 202 (async) là ok |
| R2xx-03 | patch-partial | WARN | 4 | 3 | 7 | `swap-put-patch` | PATCH nhận full DTO; PATCH nhận `Map<String,Object>`; PUT nhận full là ok |
| R2xx-04 | delete-no-body | WARN | 3 | 3 | 6 | `body-into-delete` | DELETE + `@RequestBody List<String>`; DELETE + query ids là ok |
| R2xx-05 | custom-method-post | INFO | 3 | 2 | 5 | `action-get-swap` ◇ | hành động qua GET; `/{id}:archive` AIP-136 là ok; `/export` GET là vi phạm méo |
| R2xx-06 | put-full-update | WARN | 4 | 3 | 7 | `swap-put-patch` | PUT nhận partial (tên hàm `partialUpdate*`); PUT nhận entity full là ok |
| R3xx-01 | list-paginated | WARN | 3 | 3 | 6 | `drop-pagination-param` | list không `Pageable`; list có `page,size` là ok; list trả `List<>` của projection |
| R3xx-02 | list-envelope | INFO | 3 | 2 | 5 | `unwrap-envelope` ◇ | mảng trần `List<X>`; `Page<X>` là ok; `Set<X>` nhỏ là vi phạm méo |
| R3xx-03 | unbounded-page-size | INFO | 3 | 2 | 5 | `remove-size-cap` ◇ | `size` không `@Max`; có `@Max(100)` là ok; default `size=1000` trong code |
| R3xx-04 | pagination-consistency | WARN | 4 | 3 | 7 | `split-pagination-style` | `Page<>` vs `PageResponse` trong **1 file 2 endpoint**; multi-endpoint ok-case |
| R4xx-01 | no-entity-in-payload | ERROR | 4 | 3 | 7 | `entity-into-response` | trả entity `@Entity`; entity trong generic `Page<Youth>`; DTO trùng tên entity là ok |
| R4xx-02 | field-casing | WARN | 3 | 3 | 6 | `snake-case-field` | field `created_at`; field `createdAt` là ok; acronym `userURL` (vi phạm méo) |
| R4xx-03 | time-field-standard | INFO | 3 | 2 | 5 | `time-field-to-string` ◇ | `Instant`→`String`; `OffsetDateTime` ok; epoch-long (vi phạm méo) |
| R4xx-04 | dto-naming-convention | INFO | 3 | 3 | 6 | `dto-suffix-mix` ◇ | trộn `*DTO` + `*Request`; nhất quán 1 quy ước là ok |
| R5xx-01 | unified-error-shape | WARN | 4 | 2 | 6 | `error-map-handler` | handler trả `Map`; handler trả ProblemDetail là ok; handler trả `ErrorResponse` (1 lỗi lẻ) |
| R5xx-02 | no-500-for-business | ERROR | 4 | 3 | 7 | `business-exception-500` | business exception → 500; exception tên mơ hồ (`ProcessingException`) là ok theo heuristic |
| R5xx-03 | status-semantics | WARN | 3 | 2 | 5 | `remove-status-mapping` (phụ) | phủ từ phía status; cặp với R2xx-02 — kiểm tra không double-count |
| R5xx-04 | single-error-advice | WARN | 4 | 3 | 7 | `add-second-advice` | 2 `@ControllerAdvice`; 1 advice + 1 handler cụ thể là ok |
| R6xx-01 | versioned-path | WARN (TẮT) | 3 | 2 | 5 | `unversioned-path` | path thiếu `/v1`; có `/v1` là ok — **chỉ chạy khi bật qua config** (§3.7) |
| R6xx-02 | grpc-standard-methods | INFO | 3 | 3 | 6 | `rename-rpc-nonstandard` ◇ | rpc `FetchYouthData`; `Get/List/Create/Update/Delete` là ok; service comment-out phải bỏ qua |
| R6xx-03 | version-consistency | INFO | 3 | 2 | 5 | `bump-one-path-version` ◇ | 1 endpoint `/v2` giữa bầy `/v1`; không version nào là ok (tránh đụng R6xx-01) |

**Tổng: 161 case** (91 vio + 70 ok) — ≥ 150 ✓. Mỗi rule ≥ 3 vio + ≥ 2 ok ✓. Ký hiệu ◇ = mutator OPTIONAL (§4.1).

### 3.4. Lý do phân bổ — xếp hạng họ rule theo FP tiềm năng

Nguyên tắc charter §7: ưu tiên số 1 của W4/W5 là FP. Phần dư 31 case dồn về họ có heuristic "đoán ngữ
nghĩa từ bề mặt" nhiều nhất, tỷ lệ với mật độ điểm đau w1-03:

| Hạng | Họ | Phần dư | Vì sao FP tiềm năng cao |
|---|---|---|---|
| 1 | R1xx | +10 | Parse chuỗi path tự do bằng heuristic (verb trong danh từ, casing lẫn); w1-03 pain 1–3 cho thấy mật độ verb/casing bất thường rất cao; có rule ERROR (R1xx-02) → FP phá niềm tin CI ngay |
| 2 | R2xx | +8 | Heuristic đọc ý định từ kiểu tham số/trả về (`patch-partial`, `put-full-update` đoán full/partial); 1 rule ERROR (R2xx-01); w1-03 pain 7 |
| 3 | R5xx | +5 | Phân tích chéo nhiều file (`@ControllerAdvice` chồng) + heuristic tên exception (R5xx-02 ERROR); w1-03 pain 5 |
| 4 | R4xx | +4 | Heuristic nhận entity (`@Entity`/package pattern) + suy casing JSON từ tên field; 1 rule ERROR (R4xx-01); w1-03 pain 6 + §2.4 |
| 5 | R3xx | +3 | Consistency chéo file + suy validation (`@Max`); w1-03 pain 4 |
| 6 | R6xx | +1 | R6xx-01 opt-in (FP thấp vì off-by-default); rủi ro chính là **FN** từ regex proto (w3-06) chứ không phải FP |

### 3.5. Corner cases toàn cục — `_global/` (10 case)

Ngoài case per-rule, 10 case toàn cục phủ các góc mà w4-01 deliverable 3 yêu cầu. Mỗi case khai rule
chính trong manifest (hoặc `expect: parse-ok` nếu chỉ kiểm engine không sập):

| Case (`_global/<tên>.java`) | Rule chính | Kỳ vọng |
|---|---|---|
| `nested-generic-wildcard` | R4xx-01 | `PageResponse<List<? extends YouthProjection>>` — parse + rule chạy đúng trên generic lồng |
| `wildcard-request-mapping` | R1xx-01 | `@RequestMapping("/api/*/books")` — inventory 1 endpoint, không crash |
| `overload-same-path` | R2xx-03 | 2 method cùng path khác tham số — engine không nhân đôi finding |
| `meta-annotation-composed` | R2xx-01 | custom `@MyGet` meta-annotated `@GetMapping` — **expected-miss** (v0.1 chỉ đọc annotation trực tiếp, charter A-7) |
| `multiple-base-paths` | R6xx-03 | class-level `@RequestMapping({"/a/v1", "/b"})` — inventory 2 endpoint |
| `uppercase-path-variable` | R1xx-04 | `/{YouthId}` — biến path hoa |
| `commented-out-mapping` | R1xx-02 | mapping trong comment **không** phải API surface → 0 finding (w1-03 §2.3) |
| `proto-commented-service` | R6xx-02 | `.proto` service bị comment — bỏ qua (w1-03 youth.proto:18) |
| `nested-class-controller` | R1xx-01 | controller là inner class — parse + inventory đúng |
| `no-api-plain-service` | (null) | file service thuần, không endpoint → exit 0, summary "no API surface" |

### 3.6. Manifest schema (`testdata/adversarial/manifest.yaml`)

Nguồn sự thật duy nhất; harness w4-02 đọc trực tiếp. Mỗi entry:

```yaml
- id: update-draft-path              # = slug, duy nhất trong rule
  rule: R1xx-02
  file: R1xx-02/vio-update-draft-path.java   # relative tới testdata/adversarial/
  expect: hit                        # hit = rule chính PHẢI báo ≥ 1 | miss = KHÔNG được báo | parse-ok = chỉ cần engine parse
  tier: near-miss                    # baseline | near-miss
  config: default                    # default | enable-r6xx-01
  note: >-
    POST /update/draft — động từ "update" ngụy trang action-phrase hai tầng;
    rule phải flag segment "update".
```

Trường bắt buộc: `id, rule, file, expect, tier`. Optional: `also_expect` (list rule phụ được phép/kỳ vọng
hit — dùng khi 1 case đụng 2 rule, ví dụ case R5xx-03/R2xx-02), `expected_miss: true` + `miss_reason`
(khai **trước khi chạy** khi biết rule chưa bắt được — trung thực, không phải che số), `config` (mặc định
`default`). Case `_global/` dùng `rule: <chính>` như thường, hoặc `expect: parse-ok` + `rule: null` cho
case chỉ kiểm engine.

### 3.7. Quy tắc vận hành corpus

1. **R6xx-01 mặc định TẮT** (charter A-3): corpus check chạy 2 pass — pass 1 config default (mọi thư mục
   trừ `R6xx-01/`), pass 2 bật `rules.include: [R6xx-01]` chỉ cho `R6xx-01/`. Manifest ghi `config: enable-r6xx-01`.
2. **Adjustment trong họ**: w4-01 được dồn ≤ 20% số case giữa các rule **trong cùng họ** khi viết case
   thực tế (case khó hơn dự kiến / heuristic trùng), với điều kiện: tổng ≥ 150, mỗi rule ≥ 3 vio + ≥ 2 ok,
   và bảng phân bổ thực tế chốt lại trong `manifest.yaml` + ghi lý do 1 dòng vào report w4-01. Ra ngoài
   giới hạn này → quay lại task w1-05 (orchestrator), không tự bẻ plan.
3. **Case không viết được chất lượng** (heuristic quá mơ hồ để viết near-miss): giảm về minimum 3/2 và
   ghi vào report — **không bơm case yếu cho đủ số** (anti-gaming §9).
4. **Không sửa case sau khi thấy kết quả scan** để tăng agreement — chỉ được thêm `expected_miss` kèm lý
   do kỹ thuật, và phải ghi vào report.

---

## 4. Mutation plan (w4-02)

### 4.1. Danh mục mutator

Bộ MUST (17) — w4-02 phải có đủ trước khi đo catch-rate; tên in đậm giữ nguyên tên brief w4-02.

| # | Mutator | Thao tác trên ok-case | Rule mục tiêu (phụ) |
|---|---|---|---|
| 1 | `inject-verb-path` ✓ | Thêm segment động từ vào path (`/books/{id}` → `/books/delete-item/{id}`) | R1xx-02 (R1xx-03) |
| 2 | `singularize-path` ✓ | `/books` → `/book` | R1xx-01 |
| 3 | `flatten-resource-path` | `@GetMapping("/{id}")` → `@GetMapping("/find-by-id")` | R1xx-03 |
| 4 | `mis-case-path` | `/search/all-by-condition` → `/search/all-by-Name` | R1xx-04 |
| 5 | `body-into-get` | Thêm param `@RequestBody` vào GET | R2xx-01 |
| 6 | `verb-swap` ✓ | `@GetMapping` → `@PostMapping` (endpoint đọc thành POST không 201) | R2xx-02 (R5xx-03) |
| 7 | `remove-status-mapping` ✓ | Bỏ `@ResponseStatus(CREATED)` / đổi ResponseEntity 201 → 200 | R2xx-02 (R5xx-03) |
| 8 | `body-into-delete` | Thêm `@RequestBody List<String> ids` vào DELETE | R2xx-04 |
| 9 | `swap-put-patch` | Đổi `@PatchMapping` ↔ `@PutMapping` kèm thân phương thức tương ứng | R2xx-03 (R2xx-06) |
| 10 | `drop-pagination-param` ✓ | Bỏ `Pageable` / `page,size` khỏi list endpoint | R3xx-01 |
| 11 | `split-pagination-style` | Đổi 1 trong 2 endpoint trong file từ `Page<>` sang kiểu khác | R3xx-04 |
| 12 | `entity-into-response` ✓ | Return `BookDto` → `Book` (entity `@Entity`) | R4xx-01 |
| 13 | `snake-case-field` ✓ | Field DTO `createdAt` → `created_at` | R4xx-02 |
| 14 | `error-map-handler` | `@ExceptionHandler` trả `Map<String,Object>` thay vì shape chuẩn | R5xx-01 |
| 15 | `business-exception-500` | Handler của business exception → 500 | R5xx-02 |
| 16 | `add-second-advice` | Thêm `@ControllerAdvice` bọc envelope thứ hai (non-public class cùng file) | R5xx-04 |
| 17 | `unversioned-path` ✓ | Bỏ `/v1` khỏi base path | R6xx-01 |

Bộ OPTIONAL (8, ◇ ở §3.3) — làm khi còn budget, mỗi rule thiếu mutator báo `no-mutator` + lý do trong JSON:
`mix-id-field` (R1xx-05), `action-get-swap` (R2xx-05), `unwrap-envelope` (R3xx-02), `remove-size-cap`
(R3xx-03), `time-field-to-string` (R4xx-03), `dto-suffix-mix` (R4xx-04), `rename-rpc-nonstandard`
(R6xx-02), `bump-one-path-version` (R6xx-03).

Phủ rule: MUST 17 mutator phủ 18/26 rule (mọi rule ERROR + mọi rule WARN); OPTIONAL phủ nốt 8 rule INFO.

### 4.2. Ngưỡng + định nghĩa

- **Mutation case** = 1 cặp (ok-case, mutator áp dụng). Case hợp lệ (`valid`) nếu sau mutation file vẫn
  parse được; cú pháp hỏng → loại khỏi mẫu số, log vào `broken_count` (protocol w4-02).
- **Hit** = vanguard báo ≥ 1 finding từ rule mục tiêu (tính theo JSON output; `also_expect` không tính).
- **catch-rate(rule)** = Σ hit / Σ valid của mọi mutation case nhắm rule đó. **catch-rate tổng** = Σ hit / Σ valid toàn bộ.
- **Gate**: tổng ≥ 90% (charter §7); per-rule: rule có ≥ 1 valid case thì > 0% (w4-02 acceptance).
  Per-mutator chỉ báo cáo, không gate trong v0.1 — dữ liệu postmortem.

### 4.3. Blind-spot per rule — cách đo

1. Ma trận (rule × mutator): mỗi ô ghi `valid / hit`. Ô có valid > 0 và hit = 0 → **blind spot**.
2. Mỗi blind-spot xuất 1 entry JSON: `{rule, mutator, fixture, hypothesis}` với hypothesis ∈
   `parser-gap` (adapter không thấy pattern sau mutation) | `predicate-gap` (thấy nhưng predicate
   không kích) | `config-off` (rule bị tắt/mặc định tắt).
3. Blind-spot report trong `reports/w4-02.md` group theo rule, mỗi entry 1 dòng đề xuất cải tiến
   (fix-rule / new-case / heuristic redesign) — đầu vào trực tiếp cho postmortem w5-06.

### 4.4. Ngân sách CI ≤ 5 phút — công thức chọn case

- Mỗi ô (rule, mutator) dùng **tối đa 3 ok-case** của rule mục tiêu, chọn deterministic: 3 file đầu theo
  sort tên file tăng dần (không random, bảo đảm 2 lần chạy cùng bộ).
- Ước tính: 17 mutator × ~3 case × ~0.5s/case (scan file đơn) + overhead harness ≈ < 2 phút. Nếu vượt
  4.5 phút đo thực tế → giảm 3 → 2 → 1 theo thứ tự mutator OPTIONAL trước (ghi công thức cuối cùng vào report).
- Bench repo KHÔNG dùng cho mutation (bẫy w4-02 — corpus là đủ, giữ CI nhanh).
- Mutated fixture sinh vào thư mục temp **không commit**; chỉ commit mutator code + JSON kết quả.

---

## 5. Perf & determinism protocol (w4-03)

### 5.1. Bench repo

- Generator `tools/gen-bench-repo/` (commit vào repo) — sinh **deterministic theo seed** (cùng seed = cùng
  byte). Đích: ≥ 10.000 file `.java`, trộn: controller REST (mỗi file ~5 endpoint, phủ đủ 6 họ pattern),
  service, DTO/entity, file nhiễu phi-API (util, POJO) — không file rác vô nghĩa (đều parse được).
- Generated repo **không commit** (quá lớn) — chỉ commit generator + công thức sinh + seed mặc định.

### 5.2. Môi trường + số runs

- Sandbox giống môi trường đo w1-03: image dind, **2 cpu / 4 Gi**, ttl ≥ 1 giờ (bẫy w4-03 — bench có thể > 10 phút).
- Ghi vào report: image, cpu/mem, số file thực tế (`find … | wc -l`), LOC (`wc -l`), seed, version binary (SHA).
- **5 runs** full scan (tối thiểu 3 nếu thiếu thời gian — phải ghi rõ số runs); ghi raw time từng run, không chỉ số tổng hợp.

### 5.3. Đích số (chốt bằng công cụ có sẵn — §8)

| Chỉ số | Đích | Cách tính |
|---|---|---|
| p95 wall-clock (full scan 10k file) | **≤ 60s** | nearest-rank trên ≥ 5 runs (n=5 → p95 = run chậm nhất — bảo thủ); ghi kèm raw 5 số |
| p99 | báo cáo, không gate | với n ≤ 20, p99 ≈ max — ghi rõ giới hạn thống kê, đừng giả vờ chính xác |
| Throughput | ≥ 170 file/s (suy từ 10k/60s) | files / thời gian run; báo cáo per-run |
| Memory peak | báo cáo (không gate v0.1) | chuỗi fallback §8 — nếu mọi nguồn không đo được, ghi "không đo được + lý do" |
| Thời gian start binary | ≤ 3s (charter §5.1 R4) | đo `vanguard version` cold start 3 lần |

Nếu thực đo lệch lớn (vd p95 = 120s): **không sửa ngưỡng trong task w4-03** — ghi nguyên bản + phân tích
nguyên nhân (tree-sitter query? IO?) + đề xuất; điều chỉnh ngưỡng phải qua §10.

### 5.4. Idempotency / determinism — ma trận 6 cặp

Điều kiện tiên quyết: output **không chứa** wall-clock timestamp, duration, đường dẫn tuyệt đối, hostname,
PID (renderers w2-05/w2-06 phải tuân; golden tier 2.1 cũng kiểm). SARIF mock §6.8 không có trường thời gian.

| # | Nguồn | Format | So sánh |
|---|---|---|---|
| 1 | corpus | pretty | sha256 run-1 = run-2 |
| 2 | corpus | json | idem |
| 3 | corpus | sarif | idem |
| 4 | bench repo | pretty | idem |
| 5 | bench repo | json | idem |
| 6 | bench repo | sarif | idem |

6/6 identical → pass. Fail → nguyên nhân gần như luôn nằm ở map không ổn định (sort) hoặc thời gian
được nhét vào output — sửa ở engine/render (đây là bug engine, protocol w4-03), không "chuẩn hoá" trong harness.

### 5.5. Edge cases + hành vi đã định nghĩa (không crash)

| Case | Hành vi bắt buộc |
|---|---|
| File java ≥ 5 MB | parse xong hoặc skip có diagnostics `--verbose`; exit code bình thường; không OOM |
| Nesting class sâu 10+ | parse + scan bình thường |
| Repo 0 file java hợp lệ | exit 0 + summary thông báo rõ (charter §6.3) |
| Symlink vòng trong root | walker không loop, mỗi file duyệt 1 lần |
| Filename unicode | scan bình thường, output UTF-8 nguyên vẹn |
| File java rỗng / chỉ comment | bỏ qua lặng lẽ, không finding |

Mỗi case ghi: exit code thực tế + số finding + nhận xét 1 dòng. Phân định với robustness w4-05: edge case
ở đây là **đầu vào hợp pháp ở mức cực đoan** (đo đúng/sai); đầu vào **độc hại** (path traversal `../`,
symlink chỉ ra ngoài root, config glob khổng lồ, zip-bomb-ish) là sân của w4-05 — ranh giới ghi ở bảng §2.

---

## 6. Real-fire criteria (W5)

### 6.1. Chuỗi tiêu chí theo task

| Task | Tiêu chí pass |
|---|---|
| w5-01 | SOP pull tái chạy được bởi agent khác; `vanguard version` OK; env metadata đủ (image, cpu, ttl, checkout SHA) |
| w5-02 | Scan full 3 format: số finding khớp nhau giữa 3 format; không crash; artifacts `artifacts/w5-02/` đầy đủ; timing từng lần ghi trong report; (bonus) 2 lần chạy json byte-identical trên repo thật |
| w5-03 | Recall discovery ≥ 95% theo §6.2; bảng đối chiếu endpoint đầy đủ; mỗi endpoint miss có giả thuyết nguyên nhân; bằng chứng swagger (steel screenshot) lưu workspace |
| w5-04 | 100% finding có verdict theo §7; mỗi FP có lập luận + nguồn aip.dev khi tranh chấp; tuning chỉ là **đề xuất** (không tự sửa rule) |
| w5-05 | Baseline đọc hiểu trong 5 phút; mỗi số truy vết về artifacts w5-02/03/04; backlog ≥ 5 mục cụ thể |

### 6.2. Công thức recall discovery (w5-03)

```
recall = |endpoints(swagger) ∩ endpoints(IR)| / |endpoints(swagger)|
```

- **Nguồn chuẩn đối chứng** (theo thứ tự ưu tiên): (1) springdoc JSON runtime mở qua steel (`profile api-docs`);
  (2) nếu không chạy được runtime — openapi json/annotations trong source, **ghi rõ giới hạn phương pháp** trong report.
- Endpoint = cặp `(http-method, path)` chuẩn hoá (biến path giữ nguyên dạng khai báo; sort trước khi so).
- 185 endpoint > 20 → **đếm bằng script**, script commit vào repo (`tools/`), kèm lệnh tái chạy trong report. Không đếm tay.
- gRPC (88 rpc) đối chiếu riêng theo declared service/rpc proto ↔ IR; không gộp vào số recall REST (kênh khác).
- Phạm vi khớp: cùng method + path khớp bình thường hoá biến (`{id}` ↔ `{id}`; template springdoc
  `modules/{id}` = IR `modules/{id}`). Sai lệch chuẩn hoá ghi vào report chứ không nhét vào recall.

### 6.3. Số liệu tổng hợp W5

- `reports/w5-04.md` — triage sheet theo §7; cập nhật `testdata/audit-results.json` với verdict thật.
- `reports/w5-05.md` + `baseline-military-youth.md` — số liệu cho team, truy vết về artifacts.
- `reports/w5-06.md` — postmortem: FN từ recall + blind-spot mutation + FP từ triage → danh sách
  adjustment ưu tiên, mỗi mục có tiêu chí nghiệm thu đo được (metric W4/W5 nào chứng minh nó tốt hơn).

---

## 7. Giao thức FP/FN audit

> Mục này là **nguồn chuẩn mực**; w4-04 materialize thành `docs/fpfn-protocol.md` (không đổi nội dung,
> chỉ bổ sung sample set + sheet thực tế).

### 7.1. Định nghĩa verdict

| Verdict | Định nghĩa |
|---|---|
| **TP** | Finding trùng vi phạm thật theo mô tả rule trong charter §3 — đúng rule, đúng span, đúng ngữ cảnh |
| **FP** | Code không vi phạm theo charter §3. Ba nhánh: (a) code đúng chuẩn AIP/convention; (b) code theo convention riêng của app mà **config nên tune** chứ rule không sai; (c) đúng vị trí nhưng span/message sai lệch nghiêm trọng |
| **FN** | Vi phạm thật tồn tại trong mẫu mà scanner không báo. Nguồn FN hợp lệ: corpus vio bị miss; sample audit được verifier đọc thủ công và thấy vi phạm; (riêng discovery-miss swagger là chỉ số riêng §6.2, không trộn) |
| **NE** | Chưa đánh giá được trong phiên (thiếu context). **Không được tồn dư** ở sheet cuối — w5-04 yêu cầu 100% verdict |

### 7.2. Ai label — phân vai bắt buộc

| Giai đoạn | Người label | Ràng buộc |
|---|---|---|
| W4 audit (w4-04) | **Người duyệt gate CH3** — agent chỉ sinh sheet + harness | Agent w4-04 **cấm** tự label rồi tự sửa rule cho khớp (sao chép đáp án — protocol w4-04). Sheet phải đủ rõ để người label không cần hỏi thêm |
| W5 triage (w5-04) | Verifier agent label theo tiêu chí 7.1 + reason-code, **tuning chỉ đề xuất** | Cấm sửa rule code trong task; chuẩn mực thuộc gate CH3/CH4 |
| Tranh chấp | Gate person quyết (7.4) | Agent chỉ chuẩn bị dẫn chứng |

### 7.3. Cách label — sheet format

Sheet markdown 1 finding 1 dòng (`reports/w4-04-labeling-sheet.md`, sinh tự động từ audit harness:
chạy vanguard trên sample → merge với sample metadata):

```
| # | rule | file:line | message (rút gọn) | verdict | reason-code | lý do (≤1 câu) | aip-ref |
```

- **Reason-code FP**: `heuristic-context` (heuristic sai ngữ cảnh) · `app-convention` (đúng convention
  riêng app — cần tune config) · `wrong-span` (đúng rule sai vị trí/thông điệp) · `severity-mismatch`
  (vi phạm kỹ thuật nhưng không ảnh hưởng contract — nên hạ severity).
- **Reason-code FN**: `parser-gap` · `predicate-gap` · `config-off` · `expected-miss` (đã khai trong corpus manifest).
- Finding trùng lặp (cùng rule + span) gộp 1 dòng; 1 dòng có thể gắn ≥ 1 rule nếu 2 rule cùng báo một span (ghi cả hai).
- Mẫu label ≥ 30 endpoint đại diện (`testdata/audit-sample/`, pattern mô phỏng — không copy military-youth).

### 7.4. Xử lý tranh chấp

Tranh chấp = label FP/FN mà hai bên đọc khác nhau về cùng 1 finding. Quy trình 3 bước:

1. **Tra hợp đồng nội bộ trước**: mô tả rule trong charter §3 (đã được duyệt) là phát biểu chuẩn — nếu
   charter rõ, charter thắng.
2. **Tra aip.dev gốc** (webcrawl) theo cột "AIP" của rule trong charter §3.1–3.6 — trích **nguyên văn câu**
   quyết định + URL vào cột `aip-ref` (vd `https://google.aip.dev/131` — "Standard methods: Get…"). Lưu ý:
   rule vanguard phát biểu lại bằng khái niệm Spring (charter §3.0), nên khi AIP nói về proto mà rule nói
   về Spring, ghi rõ phép quy chiếu trong ô lý do.
3. **Vẫn lưỡng lự** → đánh dấu `disputed` + đưa vào dispute log (5 điển hình vào report w4-04). Gate person
   quyết cuối. **Luật tie-break mặc định — FP-first (charter §7)**: khi không chắc code có sai hay không,
   label FP và đề xuất tune/hạ severity; không label TP "cho chắc". FP oan uổng làm chết niềm tin CI nhanh
   hơn FN bỏ lỡ.

### 7.5. Số liệu tổng hợp ở đâu

| Kho | Nội dung | Ai ghi |
|---|---|---|
| `testdata/audit-results.json` | per-rule: `{tp, fp, fn, precision, recall}` + disputes + meta (vanguard SHA, sample size). `precision = tp/(tp+fp)`; recall-audit `= tp/(tp+fn)`; chia 0 → `n/a` | w4-04 harness (sinh), w5-04 (cập nhật verdict thật) |
| `reports/w4-05.md` | Tổng hợp GATE CH3 — đối chiếu ratchet level 2: **FP < 15% (tổng)** ∧ catch-rate ≥ 90% ∧ 0 critical robustness | w4-05 |
| `reports/w5-04.md` / `w5-06.md` | Triage thật military-youth + postmortem điều chỉnh | w5-04 / w5-06 |

Phân biệt 3 chỉ số recall — không bao giờ gộp chung trong một bảng: **recall-discovery** (endpoint thấy
được, ≥ 95%, §6.2) · **catch-rate mutation** (≥ 90%, §4.2) · **recall-audit** (vi phạm thật trên mẫu
người label — báo cáo, không gate v0.1 vì phụ thuộc độ phủ sample).

---

## 8. Hộp công cụ đo

Mọi metric trong plan này đo bằng lệnh có sẵn trong sandbox (python:3.11-slim / dind image — không cài
thêm gì). Bảng này là phần trả lời cho ràng buộc "đừng đề xuất đo cái mà không có cách đo":

| Metric | Lệnh chính | Fallback / ghi chú |
|---|---|---|
| Wall-clock time | `TIMEFORMAT='%R'; time ./vanguard scan <root> --format json -o /tmp/out.json` (bash builtin) | `date +%s%N` trước/sau trong script python3 (`time.monotonic`) |
| Số file | `find <root> -name '*.java' | wc -l` | `wc -l` tổng LOC: `find … -exec wc -l {} + | tail -1` |
| Byte-identical | `sha256sum a.json b.json && cmp a.json b.json` | `diff -q` cho dạng text nhanh |
| Memory peak | `/usr/bin/time -v ./vanguard …` (nếu image có GNU time) | `cat /sys/fs/cgroup/memory.peak` (cgroup v2) → `docker stats --no-stream` (từ host dind) → nếu cả ba không có: ghi "không đo được" + lý do |
| Corpus agreement | script `tools/corpus-check.py` (w4-01 viết): đọc manifest + scan json → % khớp | kết quả in stdout + ghi JSON |
| Recall endpoint | script đếm trong `tools/` (w5-03 viết) — bắt buộc vì 185 > 20 endpoint | bảng đối chiếu CSV trong report |
| Mutation determinism | chạy `make mutation` 2 lần, `cmp` 2 JSON | sai khác bất kỳ → harness không deterministic, sửa trước khi đo |
| Cold start | `time ./vanguard version` × 3 | charter R4: ≤ 3s |

Quy tắc ghi số: **mọi số liệu trong report kèm lệnh tái chạy** (chuẩn w1-03 — không mệnh đề cảm tính);
timing dind bị nhiễu → ≥ 3 runs, ghi config sandbox, không so sánh khác config (bẫy w4-03).

---

## 9. Quy tắc anti-gaming

1. **Expected-miss khai trước khi chạy** (manifest `expected_miss` + `miss_reason`) — không thêm sau khi
   thấy kết quả xấu; thêm muộn phải ghi vào report với lý do kỹ thuật (w4-01).
2. **Số liệu nguyên bản**: catch-rate, p95, precision… ghi đúng số đo được — kể cả xấu (protocol w4-02:
   "trung thực, đừng tinh chỉnh kỳ vọng sau khi thấy kết quả"; PROCESS anti-pattern #2).
3. **Không viết case/ngưỡng để làm đẹp số**: case khó mà rule miss → dữ liệu postmortem (w4-01), không bỏ case.
4. **Không đếm tay số liệu tổng hợp** — precision/recall/agreement tính từ sheet/JSON bằng harness (w4-04).
5. **Mỗi số có nguồn**: report nào trích số phải dẫn artifact/JSON/lệnh tái chạy (w4-05).
6. **Tách người đo và người bị đo**: implementer không tự label rule của mình để qua gate (§7.2).
7. **Determinism không được "vá" ở harness**: lệch output = bug engine/render, sửa tận gốc (§5.4).

---

## 10. Điều kiện sửa plan này

- Plan này là **contract W1** — sửa sau khi w1-05 DONE phải qua orchestrator (charter §8: amendment không
  sửa lén trong task implementation), với entry log ở cuối file này.
- Sửa hợp lệ khi: (a) thực đo lệch đích số một cách có căn cứ kỹ thuật (kèm nguyên bản số cũ/mới + lệnh đo);
  (b) charter thay đổi taxonomy (thêm/bớt rule → bảng §3.3 cập nhật theo); (c) bài học real-fire W5 đòi hỏi
  ngưỡng mới cho wave-2.
- Các task không được tự thay: quy ước testdata (§3.1), schema manifest (§3.6), bộ mutator MUST (§4.1),
  ma trận idempotency (§5.4), quy trình label (§7).

---

## Log thay đổi

| Ngày | Phiên bản | Thay đổi | Ai |
|---|---|---|---|
| 2026-10-08 | v1 | Bản đầu — w1-05 | w1-05 (architect) |

*Hết test strategy v1. Charter §7 chốt đích; document này chốt cách chứng minh — mọi số đo được bằng
công cụ trong §8, mọi tầng có gate, mọi kết quả xấu vẫn là kết quả trung thực.*
