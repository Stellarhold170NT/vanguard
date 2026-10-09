// corpus: R3xx-02 vio list-bare-list
// lure: a paginated handler that still answers a bare List — no room for total/page metadata.
package com.example.books;

import java.util.List;

import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookController {

    @GetMapping("/books")
    public List<BookDto> list(Pageable pageable) {
        return List.of();
    }
}

record BookDto(Long id, String title) {
}
