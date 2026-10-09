// audit-sample: the envelope-split controller (pattern w1-03 pain 4 —
// the PageResponse wrapper beside Spring Page): custom pagination params
// without a size cap, a bare-array archived list, the byte[] export that
// stays silent everywhere.
package com.youthunion.audit.web.rest;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import com.youthunion.audit.dto.PageResponse;
import com.youthunion.audit.dto.ReportDto;
import com.youthunion.audit.dto.ReportSummaryDto;

@RestController
@RequestMapping("/api/youth-union/reports")
public class ReportResource {

    @GetMapping
    public PageResponse<ReportDto> list(@RequestParam int page, @RequestParam int size) {
        return new PageResponse<>(List.of(), 0, page, size);
    }

    @GetMapping("/archived")
    public List<ReportDto> archived() {
        return List.of();
    }

    @GetMapping("/{id}")
    public ReportDto get(@org.springframework.web.bind.annotation.PathVariable Long id) {
        return new ReportDto(id, "report");
    }

    @PostMapping("/export-by-condition")
    public byte[] exportByCondition() {
        return new byte[0];
    }

    // Binds ReportSummaryDto into the surface: the mixed-convention DTO
    // (as-008/as-034/as-035/as-036) is exercisable only from an endpoint —
    // audit rules evaluate surface-bound types, orphan types stay silent
    // (w4-04 report: construct defect found by the first harness run).
    @GetMapping("/summary")
    public ReportSummaryDto summary() {
        return new ReportSummaryDto(1L, 2L, "2025-01-01T00:00:00Z", "s");
    }
}
