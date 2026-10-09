// audit-sample: auth payloads (login token + permission item).
package com.youthunion.audit.dto;

public record PermissionDto(String code, String label) {
}
