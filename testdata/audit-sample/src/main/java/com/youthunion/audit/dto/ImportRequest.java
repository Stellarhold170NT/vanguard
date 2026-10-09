// audit-sample: import request payload (the rows arrive as raw rows; the
// entity leaks on the RESPONSE side, not here).
package com.youthunion.audit.dto;

import java.util.List;

public record ImportRequest(List<String> rows) {
}
