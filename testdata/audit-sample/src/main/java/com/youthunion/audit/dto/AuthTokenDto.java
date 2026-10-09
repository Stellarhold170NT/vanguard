// audit-sample: auth payloads (login token + permission item).
package com.youthunion.audit.dto;

public record AuthTokenDto(String token, long expiresAt) {
}
