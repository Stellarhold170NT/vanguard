// audit-sample: import request + result envelope. The controller response
// spells ImportResult<YouthMember> — the entity rides a generic argument.
package com.youthunion.audit.dto;

import java.util.List;

public record ImportResult<T>(int inserted, int rejected, List<String> errors) {
}
