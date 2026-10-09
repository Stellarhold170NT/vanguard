// audit-sample: self-id style mix (pattern w1-03 §2.4 — 182 DTOs mix
// conventions; this type spells its own id movementLogId while other types
// use id — R1xx-05 reports the mix, it does not enforce one spelling).
package com.youthunion.audit.dto;

import java.time.Instant;

public record MovementLogDto(
        Long movementLogId,
        String summary,
        Instant loggedAt) {
}
