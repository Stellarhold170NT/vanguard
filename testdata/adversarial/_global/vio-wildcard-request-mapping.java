// corpus: _global vio wildcard-request-mapping
// lure: /api/*/books — a wildcard segment between literals; inventory one endpoint, do not crash.
package com.example.routes;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class WildcardController {

    @GetMapping("/api/*/books")
    public Page<BookDto> list(Pageable pageable) {
        return Page.empty();
    }
}

record BookDto(Long id, String title) {
}
