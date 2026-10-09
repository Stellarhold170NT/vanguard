// corpus: R2xx-01 ok get-pathvars-only
// lure: plain GET with a path variable — the minimal legal read.
package com.example.books;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookItemController {

    @GetMapping("/books/{id}")
    public BookDto get(@PathVariable Long id) {
        return null;
    }
}

record BookDto(Long id, String title) {
}
