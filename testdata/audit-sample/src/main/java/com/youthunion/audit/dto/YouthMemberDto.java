// audit-sample: the clean baseline DTO — camelCase fields, Instant time
// field, self id spelled "id" (the JHipster convention the app already
// follows — w1-03 §2.4 contrast).
package com.youthunion.audit.dto;

import java.time.Instant;

public record YouthMemberDto(
        Long id,
        String fullName,
        String unitCode,
        Instant createdAt) {
}
