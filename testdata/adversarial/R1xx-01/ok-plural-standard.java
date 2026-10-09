// corpus: R1xx-01 ok plural-standard
// lure: plain plural collection plus item route — the convention the app already follows.
package com.example.library;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookController {

    @GetMapping("/books")
    public List<BookDto> list() {
        return List.of();
    }

    @GetMapping("/books/{id}")
    public BookDto get(@PathVariable Long id) {
        return null;
    }
}

record BookDto(Long id, String title) {
}
