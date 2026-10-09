// audit-sample: small clean payload DTOs shared across the controllers
// (request shapes, item DTOs, auth tokens).
package com.youthunion.audit.dto;

public record YouthAutocompleteRequest(String keyword) {
}
