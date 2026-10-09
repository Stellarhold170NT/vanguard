// corpus: R6xx-01 ok empty-base
// lure: no class-level mapping — the base path is empty, the service-level version check has nothing to read.
package com.example.books;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookController {

    @GetMapping("/api/v2/books")
    public BookDto list() {
        return null;
    }
}

record BookDto(Long id, String title) {
}
