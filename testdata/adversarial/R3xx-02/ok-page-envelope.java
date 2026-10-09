// corpus: R3xx-02 ok page-envelope
// lure: Page<T> carries total/page metadata — the framework envelope the rule accepts.
package com.example.books;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookController {

    @GetMapping("/books")
    public Page<BookDto> list(Pageable pageable) {
        return Page.empty();
    }
}

record BookDto(Long id, String title) {
}
