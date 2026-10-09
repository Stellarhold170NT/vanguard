// audit-sample: the mixed-convention DTO (pattern w1-03 §2.4) — snake_case
// JSON keys, a String time field, and two reference-id spellings in one
// type. R4xx-02 / R4xx-03 / R1xx-05 all have documented predicates here.
package com.youthunion.audit.dto;

import com.fasterxml.jackson.annotation.JsonProperty;

public record ReportSummaryDto(
        @JsonProperty("reportID") Long reportId,
        @JsonProperty("branch_id") Long branchId,
        @JsonProperty("created_date") String createdDate,
        String summary) {
}
