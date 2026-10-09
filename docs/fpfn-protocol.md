# FP/FN audit protocol (w4-04 materialization of test-strategy §7)

> Nguồn chuẩn mực là **test-strategy §7** (contract W1 của w1-05). Tài liệu này **không đổi nội
> dung** — chỉ bổ sung sample set, sheet, harness và lệnh chạy thực tế của chương trình. Nếu tài
> liệu này lệch §7, §7 thắng; sửa phải qua orchestrator (test-strategy §10).

## 0. Mục đích + vai trò (§7.2 — nguyên văn)

| Giai đoạn | Người label | Ràng buộc |
|---|---|---|
| **W4 audit (w4-04)** | **Người duyệt gate CH3** — agent chỉ sinh sheet + harness | Agent w4-04 **cấm** tự label rồi tự sửa rule cho khớp (sao chép đáp án). Sheet phải đủ rõ để người label không cần hỏi thêm |
| W5 triage (w5-04) | Verifier agent label theo tiêu chí §2 + reason-code, **tuning chỉ đề xuất** | Cấm sửa rule code trong task; chuẩn mực thuộc gate CH3/CH4 |
| Tranh chấp | Gate person quyết (§4) | Agent chỉ chuẩn bị dẫn chứng |

## 1. Định nghĩa verdict (§7.1 — nguyên văn)

| Verdict | Định nghĩa |
|---|---|
| **TP** | Finding trùng vi phạm thật theo mô tả rule trong charter §3 — đúng rule, đúng span, đúng ngữ cảnh |
| **FP** | Code không vi phạm theo charter §3. Ba nhánh: (a) code đúng chuẩn AIP/convention; (b) code theo convention riêng của app mà **config nên tune** chứ rule không sai; (c) đúng vị trí nhưng span/message sai lệch nghiêm trọng |
| **FN** | Vi phạm thật tồn tại trong mẫu mà scanner không báo. Nguồn FN hợp lệ: corpus vio bị miss; sample audit được verifier đọc thủ công và thấy vi phạm; (riêng discovery-miss swagger là chỉ số riêng test-strategy §6.2, không trộn) |
| **NE** | Chưa đánh giá được trong phiên (thiếu context). **Không được tồn dư** ở sheet cuối — w5-04 yêu cầu 100% verdict |

Hai verdict vận hành thêm trong sheet w4-04 (không thuộc §7.1, phục vụ máy):

- **CLAIM-REJECTED** — chỉ trên dòng FN candidate: claim pre-registered của verifier sai (construct
  không phải vi phạm theo charter §3). Claim bị loại khỏi vũ trụ audit → không tính vào mẫu số recall.
- **DISPUTED** — tranh chấp chưa ngã ngũ theo quy trình §4 dưới. Tính là chưa resolve (harness exit 1).

## 2. Sample set

`testdata/audit-sample/` — **39 HTTP endpoint + 10 rpc trên 8 controller**, tái lập đúng các pattern
w1-03 đã đo trên military-youth (pain 1–7, §2.1–§2.4) với domain hư cấu `com.youthunion.audit`
(`/api/youth-union/…`). **Không copy code military-youth** (charter §12) — mọi file viết mới, tối giản,
chỉ mang pattern; bảng ánh xạ pattern → construct nằm trong `testdata/audit-sample/README.md`.

`expectations.json` (cùng thư mục) là **claim pre-registered**: 40 slot vi phạm kỳ vọng + 10 anchor
"phải im lặng" + 4 coverage gap đã khai (rule charter chưa đăng ký trong registry v0.1). Claim được
ghí **trước lần quét đầu tiên** (anti-gaming test-strategy §9.1) và không bao giờ được tinh chỉnh sau
khi thấy kết quả — lệch đích là dữ liệu postmortem (w5-04), không phải lý do sửa claim.

## 3. Cách label — sheet format (§7.3 — nguyên văn format)

Sheet markdown 1 finding 1 dòng — `reports/w4-04-labeling-sheet.md`, **sinh tự động từ audit harness**
(chạy vanguard trên sample → merge với expectations):

```
| # | rule | file:line | message (rút gọn) | verdict | reason-code | lý do (≤1 câu) | aip-ref |
```

- **Reason-code FP**: `heuristic-context` (heuristic sai ngữ cảnh) · `app-convention` (đúng convention
  riêng app — cần tune config) · `wrong-span` (đúng rule sai vị trí/thông điệp) · `severity-mismatch`
  (vi phạm kỹ thuật nhưng không ảnh hưởng contract — nên hạ severity).
- **Reason-code FN**: `parser-gap` · `predicate-gap` · `config-off` · `expected-miss` (đã khai trong
  corpus manifest / expectations.json).
- Finding trùng lặp (cùng rule + span) gộp 1 dòng; 1 dòng có thể gắn ≥ 1 rule nếu 2 rule cùng báo một
  span (ghi cả hai). Muốn tực riêng TP/FP theo rule trên cùng một span → tách thành 2 dòng, mỗi dòng
  một rule (harness quy verdict cho **mọi rule trên dòng**).
- Dòng có nhiều rule: verdict áp cho toàn bộ danh sách rule của dòng.

**Quy trình submit (không cần hỏi thêm):**

1. Đọc `reports/w4-04-labeling-sheet.md` (bản in/đọc).
2. Điền `gate_verdict`, `reason_code`, `note`, `aip_ref` vào `testdata/audit-labeling.csv` — cùng id
   dòng (F-xxx = finding từ scanner; id `as-xxx`/`dm-xxx` = slot từ expectations.json).
3. Chạy: `go run ./tools/audit-check -repo . -merge testdata/audit-labeling.csv`
   - exit 0 + `testdata/audit-results.json` (labeled=true) khi sheet resolve hết và FP < ngưỡng;
   - exit 1 còn dòng chưa resolve (NE/UNCLEAR/DISPUTED) hoặc FP ≥ ngưỡng ratchet;
   - exit 2 sheet lỗi/stale (sample hoặc scanner đã đổi → regenerate rồi label lại).

## 4. Xử lý tranh chấp (§7.4 — nguyên văn 3 bước)

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

## 5. Số liệu tổng hợp (§7.5 — nguyên văn schema)

`testdata/audit-results.json` — sinh bởi harness, w5-04 cập nhật verdict thật:

```json
{
  "per_rule": [ { "rule": "R1xx-02", "tp": 0, "fp": 0, "fn": 0,
                  "precision": "n/a", "recall": "n/a",
                  "claimed": 40-slot-inventory, "observed": rows-in-scan } ],
  "aggregate": { "tp": 0, "fp": 0, "fn": 0, "precision": "…", "recall": "…", "fp_share_pct": 0 },
  "labeled": false,            // w4-04-era artifact: chưa có verdict của gate
  "disputes": [], "unresolved": [], "max_fp_share_pct": 15
}
```

- `precision = tp/(tp+fp)`; `recall-audit = tp/(tp+fn)`; chia 0 → `"n/a"` (không bao giờ 0% hoặc 100% giả).
- `claimed`/`observed` là inventory (số slot claim / số row quét) — **không phải** precision/recall.
- Ratchet level 2 (GATE CH3, w4-05 đối chiếu): **FP < 15% (tổng)** ∧ catch-rate ≥ 90% ∧ 0 critical
  robustness. Harness gate mặc định `-max-fp 15`.
- Phân biệt 3 chỉ số recall — không bao giờ gộp chung trong một bảng: **recall-discovery** (endpoint
  thấy được, ≥ 95%, test-strategy §6.2) · **catch-rate mutation** (≥ 90%, §4.2) · **recall-audit** (vi
  phạm thật trên mẫu người label — báo cáo, không gate v0.1 vì phụ thuộc độ phủ sample).

## 6. Harness — `tools/audit-check`

| Lệnh | Tác dụng |
|---|---|
| `go run ./tools/audit-check -repo .` | generate: quét sample 1 lần → `testdata/audit-findings.json` (bằng chứng thô) + `testdata/audit-labeling.csv` (sheet máy) + `reports/w4-04-labeling-sheet.md` (sheet in) + `testdata/audit-results.json` (labeled=false) |
| `go run ./tools/audit-check -repo . -merge testdata/audit-labeling.csv` | merge: quét lại (determinism — lệch sheet là stale, exit 2), đọc verdict, tính P/R per rule + FP share, ghi `audit-results.json` (labeled=true) |
| `-max-fp 15` | ngưỡng ratchet FP (mặc định 15) |
| `-scan-timeout 120s` | timeout mỗi lần quét (process amendment 2 — mọi lệnh có giới hạn) |

Nguyên tắc: harness **không label, không sửa rule, không đếm tay** (anti-gaming §9.4: mọi số liệu tổng
hợp tính từ sheet/JSON bằng harness). Alignment dùng cửa sổ ±4 dòng (anchor annotation-vs-method-line);
cùng rule + cùng file + trong cửa sổ mới khớp — lệch rule thì finding không "đoạt" slot.

## 7. Bẫy đã neo trong quy trình này

- **Không tự label rồi tự sửa rule cho khớp** — quyết định thuộc gate CH3 (§2). Dry-run của agent
  (nếu có) chỉ để kiểm chứng pipeline, ghi rõ trong report và **không bao giờ** là audit result.
- **Claim ghí trước khi quét** — đổi claim sau khi thấy kết quả phải có lý do kỹ thuật trong report.
- **Không trộn 3 recall** (discovery / mutation / audit) vào một con số.
- **FP-first** khi phân vân (§4.3).
