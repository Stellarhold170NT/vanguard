// corpus: R2xx-02 ok post-action-search
// lure: POST /books/search is an action, not a create — the action lexicon keeps both status rules silent.
package com.example.books;

import org.springframework.data.domain.Page;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookSearchController {

    @PostMapping("/books/search")
    public Page<BookDto> search(@RequestBody BookFilter filter) {
        return Page.empty();
    }
}

record BookFilter(String title) {
}

record BookDto(Long id, String title) {
}
