// corpus: R3xx-04 ok page-response-everywhere
// lure: both endpoints answer the app-owned PageResponse — consistency by convention.
package com.example.audit;

import java.util.List;

import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class AuditController {

    @GetMapping("/audit-logs")
    public PageResponse<AuditLogDto> logs(Pageable pageable) {
        return new PageResponse<>(List.of(), 0);
    }

    @GetMapping("/login-attempts")
    public PageResponse<LoginAttemptDto> attempts(Pageable pageable) {
        return new PageResponse<>(List.of(), 0);
    }
}

record PageResponse<T>(List<T> items, long total) {
}

record LoginAttemptDto(Long id, String subject) {
}
