// corpus: R3xx-04 vio page-vs-page-response
// lure: two paginated lists in one file answer Page<T> and a custom PageResponse — two wire shapes.
package com.example.reports;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ReportController {

    @GetMapping("/reports")
    public Page<ReportDto> list(Pageable pageable) {
        return Page.empty();
    }

    @GetMapping("/audit-logs")
    public PageResponse<AuditLogDto> audit(Pageable pageable) {
        return new PageResponse<>(java.util.List.of(), 0);
    }
}

record PageResponse<T>(java.util.List<T> items, long total) {
}

record ReportDto(Long id, String title) {
}

record AuditLogDto(Long id, String action) {
}
