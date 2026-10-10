# Factory metrics — definitions & method (w7-03)

> Chuẩn tham chiếu cho mọi số liệu "nhà máy đo chính nó" trong VANGUARD.
> Máy sinh số liệu: `tools/factory-metrics.py` (repo) → `state/factory-metrics.json`
> (machine-readable, `schema_version=1`) + `state/factory-metrics.md` (bảng người đọc).
> Báo cáo chu kỳ mẫu dùng bộ định nghĩa này: `docs/factory-cycle-report-sample.md`.

## 1. Nguồn dữ liệu & phạm vi

| Nguồn | Đường dẫn | Trạng thái lúc sinh metrics mẫu (2026-10-10T09:17Z) |
|---|---|---|
| Task contracts | `taskloop/task-*.task.json` (41 file) | AVAILABLE — 0 parse errors |
| Factory runlog | `state/runlog.jsonl` | **KHÔNG TỒN TẠI** — orchestrator chưa ghi |
| Contract statuses | trường `status` trong contracts | AVAILABLE |

Phạm vi dữ liệu của metrics mẫu: 41 contracts, window 2026-10-08T09:48Z → 2026-10-10T09:15Z
(47.4 h), workflow W1..W7 (W7 mới 3 IN_PROGRESS + 2 PENDING). **Không diễn giải quá mức**:
đây là dữ liệu demo trên những gì có sẵn, không phải tổng kết chương trình (w7-05 mới là
báo cáo tổng).

Quy tắc bất di bất dịch: `runlog.jsonl` là nguồn chân truth append-only — script chỉ ĐỌC;
số liệu sai thì sửa script, không bao giờ sửa runlog.

## 2. Định nghĩa metric (công thức + nguồn)

Ký hiệu: `DWC` = DONE_WITH_CONCERNS. "Arm" = một lượt chạy task từ `IN_PROGRESS` đến một
trạng thái terminal (`DONE | DONE_WITH_CONCERNS | BLOCKED | NEEDS_CONTEXT`).

| Metric | Công thức | Nguồn | Có sẵn lúc mẫu? |
|---|---|---|---|
| `concerns_rate` | `DWC / (DONE + DWC)`, theo workflow và tổng; `null` khi chưa có task nào terminal DONE* | contract `status` | ✅ (tổng 30.6%) |
| `span_hours` | `last_touch − first_touch` trên `updatedAt` của contracts trong 1 workflow (first_touch của contract PENDING == lúc tạo). **Wall-clock proxy của workflow, KHÔNG phải thời gian/task** | contract `updatedAt` | ✅ |
| `avg/min/max_task_hours` | thời gian 1 arm = `ts(chuyển vào terminal) − ts(chuyển vào IN_PROGRESS)` của cùng arm đó, tính mean/min/max theo workflow | **runlog only** | ❌ `null` |
| `rearm_count` | số arm trừ arm đầu tiên của mỗi task (task chạy lại sau khi đã terminal thì +1) | **runlog only** | ❌ `null` |
| `done_first_try_rate` | (task có arm ĐẦU TIÊN kết thúc DONE) / (mọi task từng đạt DONE) | **runlog only** | ❌ `null` |
| `gates` | các comment contract có `author` chứa "gate"; quyết định = token `APPROVED|REJECTED` đầu tiên của `text`; bổ sung event `gate_result` từ runlog khi có | contracts + runlog | ✅ (5 APPROVED) |
| `reports_missing_count` | contracts khai báo `report` mà file không tồn tại trên disk lúc chạy (chỉ kiểm khi thư mục `dir` của contract nhìn thấy được — trong sandbox sẽ tự skip kèm lý do, không báo giả "missing") | filesystem | ✅ |

### Ngữ nghĩa của `null`

`null` = **chưa đo được** (nguồn chưa tồn tại). Script không bao giờ thay `null` bằng `0` —
một số 0 giả còn tệ hơn một khoảng trống trung thực. w7-04 đọc JSON phải phân biệt
`null` (chưa có dữ liệu) với `0` (có dữ liệu và bằng 0).

## 3. Ghi chú phương pháp — objective vs cộng tay (Deliverable 4)

**Đo objective từ dữ liệu có sẵn (script tự tính, truy vết được):**
`concerns_rate`, `span_hours`, `gates`, `reports_missing_count`, và (khi runlog tồn tại)
`avg/min/max_task_hours`, `rearm_count`, `done_first_try_rate`.

**Không đo được hiện tại — nếu cần phải cộng tay / xây thêm nguồn:**

1. **Thời gian thực/task so với estimate (45', v.v.)** — contract KHÔNG ghi estimate và
   thực tế; `updatedAt` chỉ cho mốc cuối. Cần runlog (schema §4) hoặc ghi `estimate_hours`
   vào contract. Ngày mẫu: ước lượng trong brief (45') không đối chiếu được với thực tế.
2. **Số lần re-arm, gates pass lần đầu** — contracts chỉ giữ trạng thái HIỆN TẬI (snapshot),
   mất lịch sử chuyển trạng thái; chỉ runlog giữ được. Ngày mẫu: các task từng
   NEEDS_CONTEXT rồi DONE vẫn hiện `DONE` — lịch sử re-arm đã mất.
3. **Cost/model usage** — `model` trong contract là model DỰ KIẾN, không phải model thực chạy,
   và không có số token/cost.
4. **Gate pass lần đầu** — comment gate không ghi đây là lần duyệt thứ mấy của workflow đó;
   nếu từng REJECTED rồi APPROVED, comment cũ có thể bị ghi đè trong contract. Chỉ runlog
   hoặc Git history của contract mới chứng minh được.

## 4. Runlog event schema (ĐỀ XUẤT cho orchestrator)

File `state/runlog.jsonl`, append-only, 1 JSON object mỗi dòng:

```json
{"ts":"2026-10-10T06:35:11Z","event":"task_status","task":"w7-03","from":"PENDING","to":"IN_PROGRESS"}
{"ts":"2026-10-10T07:02:45Z","event":"task_status","task":"w7-03","from":"IN_PROGRESS","to":"DONE"}
{"ts":"2026-10-10T07:02:45Z","event":"gate_result","task":"w6-05","to":"APPROVED"}
```

- Bắt buộc: `ts` (ISO-8601), `event`, `task`. `task` chấp nhận `w7-03`, `[w7-03] …`,
  `task-39` hoặc số contract `39` (script tự map qua contract n→workflow; id lạ được
  liệt kê trong `gates.runlog.unmapped_tasks`, không bị nuốt âm thầm).
- Bắt buộc với `task_status`: `from`, `to`. Alias key được chấp nhận: `timestamp|time|@timestamp`,
  `type|action|kind`, `task_id|taskid|name|id`.
- Dòng không hiểu được được ĐẾM (`lines_unrecognized`) chứ không làm chết script.

## 5. Hợp đồng máy đọc cho w7-04 (schema v1)

- Top-level: `schema_version`, `generated_at`, `generator`, `data_sources`, `totals`,
  `workflows[]`, `gates`, `reports_check`, `definitions`, `runlog_event_schema_proposed`.
- Mọi số phẳng ở `totals`; mọi mảng theo workflow ở `workflows[]` (khóa `workflow` = `w1`..`w7`).
- Định nghĩa metric nhúng ngay trong JSON (`definitions`) — JSON tự mô tả, không cần parse
  tài liệu riêng. Thay đổi cấu trúc phải tăng `schema_version`.

## 6. Tích hợp Grafana/Loki (Deliverable 2) — blocker & đề xuất

**Trạng thái ngày mẫu:** tools `grafana_services`/`grafana_logs` từ chối chạy với lỗi
`GRAFANA_URL / GRAFANA_API_KEY not set — add them in Settings → Secrets (vault, hot-reload)`
→ chưa có demo LogQL live trong phiên này (blocker môi trường, không phải lỗi script).

**Đề xuất pipeline khi có credential:** orchestrator append 1 dòng JSON (schema §4) vào
Loki sau mỗi chuyển trạng thái; Label đề xuất: `service_name="vanguard-factory"`,
`factory_event="task_status"`. Query mẫu (chạy sau khi có datasource):

```logql
{service_name="vanguard-factory"} | json | event="task_status"          # mọi chuyển trạng thái
{service_name="vanguard-factory"} | json | to="DONE_WITH_CONCERNS"      # Concerns stream
{service_name="vanguard-factory"} | json | event="gate_result"          # gate history
```

**Datasource picker (WIKI §6):** cụm có 2 Loki — `loki-uat` và `loki-dev`; chọn sai sẽ thấy
"0 kết quả" mà không phải vì dữ liệu thiếu. Kiểm tra datasource TRƯỚC khi kết luận
(vault `GRAFANA_LOKI_DS`); với factory, dữ liệu thuộc môi trường orchestrator đang chạy.
