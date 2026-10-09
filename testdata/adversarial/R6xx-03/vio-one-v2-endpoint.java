// corpus: R6xx-03 vio one-v2-endpoint
// lure: one /api/v2 endpoint among /api/v1 siblings — the half-finished version migration.
package com.example.books;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class VersionDriftController {

    @GetMapping("/api/v1/books")
    public Page<BookDto> list(Pageable pageable) {
        return Page.empty();
    }

    @GetMapping("/api/v2/books")
    public Page<BookDto> listV2(Pageable pageable) {
        return Page.empty();
    }
}

record BookDto(Long id, String title) {
}
