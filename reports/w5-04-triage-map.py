import json, sys, collections

SCAN = "/app/workspace/vanguard/artifacts/w5-02/scan.json"
data = json.load(open(SCAN, encoding="utf-8"))
findings = data["findings"]
assert len(findings) == 244, len(findings)

TP, FP, FN = "TP", "FP", "FN"

# --- mapping: index -> (verdict, reason_code, reason, aip) ---
M = {}

# R4xx-03 (0..10): TP all
for i in range(11):
    M[i] = (TP, "", "Field giữ giá trị thời gian bằng String/LocalDateTime — thiếu offset/RFC3339 như charter §3.4 yêu cầu; INFO đúng mức", "aip.dev/142")

# R3xx-01 (20 all TP)
R3XX01 = [11,13,15,18,21,24,36,39,58,78,117,130,134,152,161,194,199,211,215,224]
for i in R3XX01:
    M[i] = (TP, "", "GET trả list không tham số phân trang — đúng mô tả rule (dump-risk thật; bounded-vocab xử lý bằng tuning, không đảo verdict)", "aip.dev/158")

# R3xx-02 (27): TP 24, FP 3 (action responses)
for i in [12,14,16,19,22,25,31,37,40,59,64,71,79,113,118,131,135,153,162,195,200,212,216,225]:
    M[i] = (TP, "", "List<DTO/entity> trần không envelope — đúng rule (response list cần chỗ cho metadata)", "aip.dev/158")
for i in [89,122,126]:
    M[i] = (FP, "heuristic-context", "List là response của custom action (upload/approve), không phải List method — AIP-158 govern List methods; precedents w4-04 dispute #2/#3", "aip.dev/158")

# R1xx-01 (21): TP 2, FP 19
for i in [51,53]:
    M[i] = (TP, "", "Base collection 'innovation-by-unit' là danh từ số ít thật — AIP-122: 'Collection identifiers must be plural'", "aip.dev/122")
for i in [17,20,23,26,35,41,49,55,66,83,108,123,144,190,193,198,206,210,223]:
    M[i] = (FP, "heuristic-context", "Segment là hành động/chức năng (auto-complete/tree/filter/dropdown/all-by-condition/approval/recall...), không phải collection noun — AIP-122 chỉ govern collection identifiers", "aip.dev/122")

# R1xx-02 (14 all TP)
for i in [33,74,97,99,104,115,119,128,149,159,170,182,213,226]:
    M[i] = (TP, "", "Path chứa verb CRUD (delete/update/get/find) — charter pain 1+3; ERROR đúng mức vì cùng thao tác có 2 shape endpoint", "aip.dev/131")

# R1xx-03 (2 TP)
M[183] = (TP, "", "{id} nằm dưới action segment 'update-status-uncheck' — route phẳng phá hierarchy tài nguyên (AIP-127)", "aip.dev/127")
M[227] = (TP, "", "{id} nằm dưới action segment 'find-by-id' — chuẩn đối chiếu là /{collection}/{id} (AIP-127)", "aip.dev/127")

# R1xx-04 (1 TP)
M[56] = (TP, "", "'all-by-Name' hoa giữa kebab — pain 2, lệch chính convention của chính app (7 nơi /search/all-by-condition đúng)", "aip.dev/122")

# R2xx-01 (2): TP 197, FP 238
M[197] = (TP, "", "GET /auto-complete khai @RequestBody (pain 7a) — client phổ biến bỏ GET body, ERROR đúng", "aip.dev/131")
M[238] = (FP, "heuristic-context", "Test fixture (src/test, ExceptionTranslatorTestController) không phải API surface; tham số là @RequestPart chứ không phải @RequestBody — message sai phạm vi", "aip.dev/131")

# R2xx-02 (30): TP 12, FP 18
for i in [43,47,80,92,94,95,100,136,145,155,166,185]:
    M[i] = (TP, "", "POST tạo resource thật trả 200 (ResponseEntity.ok) — phải khai 201 (AIP-133; JHipster convention của chính app trả 201)", "aip.dev/133")
for i in [28,61,68,110,174,180,201]:
    M[i] = (FP, "heuristic-context", "Endpoint ĐÃ khai 201 qua ResponseEntity.created(URI) — heuristic không đọc được runtime status → message 'does not declare 201' sai sự thật", "aip.dev/133")
for i in [85,87,102,139,147,157,163,168,218,220,221]:
    M[i] = (FP, "heuristic-context", "POST là action (approval/recall/filter/getPermission/split/merge/dissolve), không tạo resource — predicate create-shaped over-fire (w4-04 dispute #1 precedent)", "aip.dev/133")

# R5xx-03 (32): TP 12, FP 20 (script xác nhận count)
for i in [44,48,81,93,96,101,107,138,146,156,167,186]:
    M[i] = (TP, "", "POST tạo resource thật trả 200 implicit — phải 201 (cặp với R2xx-02); SCC generate-code javadoc xác nhận 'will be saved'", "aip.dev/133")
for i in [29,62,69,111,175,181,202]:
    M[i] = (FP, "heuristic-context", "Endpoint ĐÃ khai 201 qua ResponseEntity.created(URI) — heuristic không thấy", "aip.dev/133")
for i in [84,86,88,103,140,141,148,158,164,169,219,222]:
    M[i] = (FP, "heuristic-context", "POST là action/query (send-approve/approval/recall/filter/getPermission/split/dissolve), không create — 200 là đúng", "aip.dev/133")
M[98] = (FP, "heuristic-context", "POST /update/draft là UPDATE bản nháp (service.updateDraft) — 200 đúng cho update (AIP-134); message 'while creating' sai ngữ cảnh", "aip.dev/134")

# R2xx-03 (4 all FP)
for i in [30,63,70,112]:
    M[i] = (FP, "app-convention", "PATCH consume merge-patch+json của JHipster = partial update by design (AIP-134: PATCH should be used cho partial); heuristic 'body type = response type' không thấy merge-patch semantics", "aip.dev/134")

# R2xx-04 (14 all TP)
R2XX04 = [34,46,54,75,76,105,120,132,150,160,171,177,188,204]
for i in R2XX04:
    M[i] = (TP, "", "DELETE khai @RequestBody List<String> — AIP-135 'must not be a body key'; xoá nhiều nên batch path/query (AIP-165)", "aip.dev/135")

# R2xx-05 (34): TP 4, FP 30
for i in [121,125,165,184]:
    M[i] = (TP, "", "Action có side effect chạy GET/PUT (sync-departments/approve/update-status) — AIP-136: 'POST must be used if the method has side effects'", "aip.dev/136")
for i in [27,32,38,42,50,57,65,67,72,73,77,90,109,114,116,124,127,129,133,154,173,178,179,191,192,207,208,209,214,228]:
    M[i] = (FP, "heuristic-context", "GET thuần đọc (export/download/search/get-term/generate-code/check-validate/find...): AIP-136 'GET must be used for methods retrieving data' → INFO verify-hint không đúng trên span này", "aip.dev/136")

# R2xx-06 (7): TP 1, FP 6
M[187] = (TP, "", "partialUpdateYouth: PUT làm partial update (tên method + pain 7c w1-03) — partial phải là PATCH (AIP-134)", "aip.dev/134")
for i in [45,52,82,143,176,203]:
    M[i] = (FP, "heuristic-context", "PUT + request type riêng (UpdateRequest) = full-replacement hợp lệ (AIP-134 cho phép PUT cho full replace); 'payload≠response → partial' là suy diễn, không đọc được semantics", "aip.dev/134")

# R4xx-01 (8 all TP)
for i in [60,137,142,151,172,189,205,217]:
    M[i] = (TP, "", "Trả ORM entity trực tiếp trong payload/response — schema DB + lazy relations leak ra wire (pain 6); ERROR đúng", "aip.dev/121")

# R6xx-02 (3 all TP)
for i in [229,230,231]:
    M[i] = (TP, "", "rpc 'PartialUpdate*' không phải Standard Method name (Get*/List*/Create*/Update*/Delete*) — INFO hợp lý; tuning: cân nhắc thêm vào lexicon VerbNoun", "aip.dev/134")

# R6xx-92 (demo) — all FP
R6XX92 = [91,106,232,233,234,235,236,239,240,241,242,243]
for i in R6XX92:
    if i in (91,106):
        M[i] = (FP, "heuristic-context", "Demo rule (ngoài 26 rule charter §3); void là chủ ý — downloadZip stream zip qua HttpServletResponse, syncRestRoutes là ops endpoint", "")
    else:
        M[i] = (FP, "heuristic-context", "Demo rule (ngoài charter §3) fire trên test fixture src/test — test controller không phải API surface", "")

# R6xx-93 (2 all FP)
M[196] = (FP, "heuristic-context", "Demo rule trùng chính xác violation mà R2xx-01 (ERROR) đã report trên cùng span — demo family không nên active trên production", "")
M[237] = (FP, "heuristic-context", "Demo rule trên test fixture src/test; tham số là @RequestPart — không phải API surface", "")

# --- validate coverage ---
missing = [i for i in range(244) if i not in M]
if missing:
    print("MISSING MAPPING:", missing)
    for i in missing:
        f = findings[i]
        print(i, f["ruleId"], f["severity"], f["location"]["file"] + ":" + str(f["location"]["line"]), f["message"][:90])
    sys.exit(1)

# --- emit table grouped by rule + counts ---
by_rule = collections.OrderedDict()
for i, f in enumerate(findings):
    v, rc, reason, aip = M[i]
    by_rule.setdefault(f["ruleId"], []).append((i, f, v, rc, reason, aip))

tot = collections.Counter()
for rule, rows in by_rule.items():
    c = collections.Counter(r[2] for r in rows)
    tot.update(c)
print("=== PER-RULE ===")
for rule, rows in by_rule.items():
    c = collections.Counter(r[2] for r in rows)
    print(f"{rule}: total={len(rows)} TP={c['TP']} FP={c['FP']} FN={c['FN']}")
print("=== AGGREGATE ===", dict(tot), "sum=", sum(tot.values()))
fp_pct = round(100.0 * tot["FP"] / 244, 1)
print(f"FP share = {fp_pct}% (ratchet <15%)")

# markdown table
out = []
out.append("| # | Rule | Sev | Vị trí | Message (rút gọn) | Verdict | Reason-code | Lý do | aip-ref |")
out.append("|---|---|---|---|---|---|---|---|---|")
for rule, rows in by_rule.items():
    for i, f, v, rc, reason, aip in rows:
        loc = f["location"]["file"].replace("src/main/java/com/viettel/tcct/", "") + ":" + str(f["location"]["line"])
        msg = f["message"]
        if len(msg) > 88:
            msg = msg[:85] + "..."
        msg = msg.replace("|", "/")
        out.append(f"| {i} | {rule} | {f['severity']} | `{loc}` | {msg} | **{v}** | {rc or '—'} | {reason} | {aip or '—'} |")
open("/app/workspace/vanguard/.worktrees-tmp-w504-table.md", "w", encoding="utf-8").write("\n".join(out) + "\n")
print("table written, rows:", len(out) - 2)
