// corpus: R3xx-01 vio list-unpaginated
// lure: a plain collection GET with no page input — one request can dump the table.
package com.example.books;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookListController {

    @GetMapping("/books")
    public List<BookDto> list() {
        return List.of();
    }
}

record BookDto(Long id, String title) {
}
