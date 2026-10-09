// corpus: R6xx-03 vio v2-among-v1
// lure: two /api/v1 endpoints and one /api/v2 — the minority version segment stands out.
package com.example.catalog;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class CatalogController {

    @GetMapping("/api/v1/books")
    public Page<BookDto> books(Pageable pageable) {
        return Page.empty();
    }

    @GetMapping("/api/v1/magazines")
    public Page<MagazineDto> magazines(Pageable pageable) {
        return Page.empty();
    }

    @GetMapping("/api/v2/loans")
    public Page<LoanDto> loans(Pageable pageable) {
        return Page.empty();
    }
}

record BookDto(Long id, String title) {
}

record MagazineDto(Long id, String issue) {
}

record LoanDto(Long id, java.time.Instant dueAt) {
}
