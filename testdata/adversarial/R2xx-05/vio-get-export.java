// corpus: R2xx-05 vio get-export
// lure: GET /reports/export — an action served by GET; AIP-136 wants POST <collection>:export.
package com.example.reports;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ReportController {

    @GetMapping("/reports/export")
    public ReportDto export() {
        return null;
    }
}

record ReportDto(Long id, String csvUrl) {
}
