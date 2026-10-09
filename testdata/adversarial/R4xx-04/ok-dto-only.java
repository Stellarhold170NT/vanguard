// corpus: R4xx-04 ok dto-only
// lure: every payload type ends in *DTO — one convention across requests and responses.
package com.example.books;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookController {

    @GetMapping("/books/{id}")
    public BookDto get(@PathVariable Long id) {
        return null;
    }

    @PostMapping("/books")
    public BookDto create(@RequestBody BookDto draft) {
        return draft;
    }
}

record BookDto(Long id, String title) {
}
