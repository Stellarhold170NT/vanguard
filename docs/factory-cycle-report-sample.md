# VANGUARD factory — báo cáo chu kỳ mẫu (cycle report sample)

> **Chuẩn format cho w7-05 trích dẫn.** Sinh từ `tools/factory-metrics.py` + contracts tại
> thời điểm 2026-10-10T09:17Z. Định nghĩa từng số: `docs/factory-metrics.md` — không số
> "thần thánh" nào ngoài các định nghĩa đó. Đây là **báo cáo chu kỳ ĐANG CHẠY** (W7 chưa xong),
> không phải tổng kết chương trình.

## 1. Tóm tắt chu kỳ (một màn hình)

- **Tiến độ DAG**: 36/41 task terminal (25 DONE · 11 DONE_WITH_CONCERNS), 3 IN_PROGRESS
  (w7-01, w7-02, w7-03), 2 PENDING (w7-04, w7-05). Không có BLOCKED/NEEDS_CONTEXT tồn đọng.
- **Concerns rate toàn nhà máy**: 11/36 = **30.6%** (W3 cao nhất: 71.4% — 5/7; W1 sạch: 0%).
- **Gates**: 5/5 APPROVED lần đầu quan sát được (w1-06, w2-07, w4-05, w5-06, w6-05 — author
  `human-gate`); 0 REJECTED. *Lưu ý: "pass lần đầu" chính xác cần runlog — contract chỉ giữ
  snapshot hiện tại (docs/factory-metrics.md §3).*
- **Wall-clock**: 2026-10-08T09:48Z → 2026-10-10T09:15Z (47.4 h), W1 gọn nhất (1.2 h span),
  W2/W3 dài nhất (~15 h span).

## 2. Bảng theo workflow (nguồn: state/factory-metrics.json schema v1)

| workflow | tasks | done | concerns | concerns_rate | in_prog | pending | span_h |
|---|---:|---:|---:|---:|---:|---:|---:|
| w1 | 6 | 6 | 0 | 0.0% | 0 | 0 | 1.2 |
| w2 | 7 | 4 | 3 | 42.9% | 0 | 0 | 14.7 |
| w3 | 7 | 2 | 5 | 71.4% | 0 | 0 | 15.6 |
| w4 | 5 | 4 | 1 | 20.0% | 0 | 0 | 5.9 |
| w5 | 6 | 5 | 1 | 16.7% | 0 | 0 | 3.4 |
| w6 | 5 | 4 | 1 | 20.0% | 0 | 0 | 7.3 |
| w7 | 5 | 0 | 0 | n/a | 3 | 2 | 47.4 |
| **total** | **41** | **25** | **11** | **30.6%** | **3** | **2** | **47.4** |

Chưa đo được (nguồn chưa tồn tại, `null` ≠ 0): avg/min/max giờ mỗi task, re-arm count,
DONE-first-try rate — chờ orchestrator ghi `state/runlog.jsonl` (schema đề xuất:
docs/factory-metrics.md §4).

## 3. Vệ sinh dữ liệu phát hiện bởi metrics

- `reports_check`: 7 contracts khai báo report chưa có ở đường dẫn contract-declared:
  w7-01..w7-05 (đang chạy/chờ — bình thường) và **w3-06, w4-04 (DONE_WITH_CONCERNS —
  report đã commit vào `repo/reports/` nhưng không nằm ở `reports/` workspace như contract
  ghi; nên đồng bộ lại đường dẫn để truy vết w7-05 không phải đi vòng)**.

## 4. Khuyến nghị cho chu kỳ kế tiếp

1. Orchestrator bật `state/runlog.jsonl` (5 dòng code, schema §4) — mở khóa 3 metric đối
   tượng: thời gian/task, re-arm, DONE-first-try. Đây là khoảng trống đo lớn nhất hiện tại.
2. Đẩy cùng dòng JSON đó lên Loki (`service_name="vanguard-factory"`) — dashboard LogQL
   sẵn trong docs/factory-metrics.md §6; nhớ datasource picker loki-uat/loki-dev.
3. Giữ concerns rate dưới ratchet PROCESS §6: W3 cho thấy nhóm rule-engine (R5xx/R6xx) là
   vùng rủi ro DONE_WITH_CONCERNS — wave kế tiếp nên tách brief nhỏ hơn cho vùng này.

---
*Sinh tự động từ contracts + (tương lai) runlog — tái tạo bằng
`python3 tools/factory-metrics.py` từ thư mục workspace; đừng sửa số bằng tay.*
