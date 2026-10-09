// corpus: R3xx-03 ok size-capped
// lure: the same size parameter with @Max — capped, silent.
package com.example.books;

import org.springframework.data.domain.Page;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import jakarta.validation.constraints.Max;

@RestController
public class BookController {

    @GetMapping("/books")
    public Page<BookDto> list(@RequestParam("size") @Max(100) int size) {
        return Page.empty();
    }
}

record BookDto(Long id, String title) {
}
