// corpus: R3xx-03 ok page-offset-params
// lure: page/offset parameters — neither is a size word, so no cap is demanded.
package com.example.audit;

import org.springframework.data.domain.Page;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class AuditController {

    @GetMapping("/audit-logs")
    public Page<AuditLogDto> list(@RequestParam int page, @RequestParam int offset) {
        return Page.empty();
    }
}

record AuditLogDto(Long id, String action) {
}
