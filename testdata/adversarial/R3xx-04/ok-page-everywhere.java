// corpus: R3xx-04 ok page-everywhere
// lure: both list endpoints answer Page<T> with a Pageable — one pagination dialect.
package com.example.books;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookController {

    @GetMapping("/books")
    public Page<BookDto> books(Pageable pageable) {
        return Page.empty();
    }

    @GetMapping("/magazines")
    public Page<MagazineDto> magazines(Pageable pageable) {
        return Page.empty();
    }
}

record BookDto(Long id, String title) {
}

record MagazineDto(Long id, String issue) {
}
