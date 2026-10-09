// corpus: R4xx-04 vio record-vs-dto
// lure: one handler answers a plain record, its sibling a *DTO — two naming spellings for the same role.
package com.example.catalog;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class CatalogController {

    @GetMapping("/authors/{id}")
    public AuthorRecord get(@PathVariable Long id) {
        return null;
    }

    @GetMapping("/books/{id}")
    public BookDto getBook(@PathVariable Long id) {
        return null;
    }
}

record AuthorRecord(Long id, String penName) {
}

record BookDto(Long id, String title) {
}
