// corpus: R3xx-03 vio size-unbounded
// lure: a size query parameter with no @Max — one client can request unbounded pages.
package com.example.books;

import org.springframework.data.domain.Page;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookController {

    @GetMapping("/books")
    public Page<BookDto> list(@RequestParam("size") int size) {
        return Page.empty();
    }
}

record BookDto(Long id, String title) {
}
