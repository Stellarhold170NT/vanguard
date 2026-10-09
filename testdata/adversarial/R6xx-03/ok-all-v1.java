// corpus: R6xx-03 ok all-v1
// lure: every endpoint pins /api/v1 — one version across the prefix.
package com.example.books;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookController {

    @GetMapping("/api/v1/books")
    public Page<BookDto> books(Pageable pageable) {
        return Page.empty();
    }

    @GetMapping("/api/v1/authors")
    public Page<AuthorDto> authors(Pageable pageable) {
        return Page.empty();
    }
}

record BookDto(Long id, String title) {
}

record AuthorDto(Long id, String penName) {
}
