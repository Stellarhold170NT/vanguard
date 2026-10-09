// corpus: R6xx-03 vio version-suffix-drift
// lure: the same service prefix serves /api/v1/books and /api/v1/books/v2 — versions drift inside one prefix.
package com.example.shelves;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ShelfController {

    @GetMapping("/api/v1/shelves")
    public Page<ShelfDto> list(Pageable pageable) {
        return Page.empty();
    }

    @GetMapping("/api/v1/shelves/v2")
    public Page<ShelfDto> listV2(Pageable pageable) {
        return Page.empty();
    }
}

record ShelfDto(Long id, String label) {
}
