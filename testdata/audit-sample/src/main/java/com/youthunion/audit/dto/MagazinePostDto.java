// audit-sample: item DTOs and the import pair. ImportResult carries the
// entity type as a generic argument in the import response (pattern w1-03
// pain 6 — ImportResult<Youth>).
package com.youthunion.audit.dto;

public record MagazinePostDto(Long id, String title) {
}
