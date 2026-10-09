// corpus: R4xx-02 ok camel-standard
// lure: all lowerCamel fields — nothing to normalize.
package com.example.books;

import java.time.Instant;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookController {

    @GetMapping("/books/{id}")
    public BookDto get(@PathVariable Long id) {
        return null;
    }
}

record BookDto(Long id, String title, Instant createdAt) {
}
