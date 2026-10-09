// audit-sample: partial-update payload (the smaller body R2xx-06's
// mismatch heuristic reads) and the patch payload for the clean PATCH.
package com.youthunion.audit.dto;

public record YouthStatusUpdate(String status, String note) {
}
