// audit-sample: single-field patch payload — type differs from the
// response type, so the clean PATCH stays silent under R2xx-03.
package com.youthunion.audit.dto;

public record YouthMemberPatch(String fullName) {
}
