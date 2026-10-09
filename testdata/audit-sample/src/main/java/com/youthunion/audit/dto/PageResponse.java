// audit-sample: app-wide pagination envelope (pattern w1-03 pain 4 — the
// private-jar PageResponse wrapper living beside Spring Page; here the
// wrapper is a plain record so the scan can see it).
package com.youthunion.audit.dto;

import java.util.List;

public record PageResponse<T>(
        List<T> items,
        long total,
        int page,
        int size) {
}
