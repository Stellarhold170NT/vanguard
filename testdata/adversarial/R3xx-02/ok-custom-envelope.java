// corpus: R3xx-02 ok custom-envelope
// lure: an app-owned PageResponse record — not a bare container, so the envelope rule stays silent.
package com.example.audit;

import java.util.List;

import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class AuditController {

    @GetMapping("/audit-logs")
    public PageResponse<AuditLogDto> list(Pageable pageable) {
        return new PageResponse<>(List.of(), 0);
    }
}

record PageResponse<T>(List<T> items, long total) {
}

record AuditLogDto(Long id, String action) {
}
