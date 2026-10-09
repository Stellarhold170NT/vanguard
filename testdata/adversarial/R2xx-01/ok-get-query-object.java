// corpus: R2xx-01 ok get-query-object
// lure: the same filter declared @RequestParam — bound from the query string, no body in sight.
package com.example.books;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookQueryController {

    @GetMapping("/books")
    public Page<BookDto> search(@RequestParam BookQuery query, Pageable pageable) {
        return Page.empty();
    }
}

record BookQuery(String title, String author) {
}

record BookDto(Long id, String title) {
}
