// corpus: R2xx-01 vio get-request-body
// lure: GET with @RequestBody next to a Pageable — the filter rides a body many clients drop.
package com.example.books;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookSearchController {

    @GetMapping("/books")
    public Page<BookDto> search(@RequestBody BookFilter filter, Pageable pageable) {
        return Page.empty();
    }
}

record BookFilter(String title, String author) {
}

record BookDto(Long id, String title) {
}
