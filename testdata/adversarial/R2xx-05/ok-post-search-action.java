// corpus: R2xx-05 ok post-search-action
// lure: the same /books/search shape under POST — one verb away from the GET violation the rule flags.
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
