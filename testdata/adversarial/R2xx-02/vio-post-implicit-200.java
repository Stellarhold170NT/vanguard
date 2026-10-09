// corpus: R2xx-02 vio post-implicit-200
// lure: POST /books returning the new BookDto with no declared status — the implicit 200 hides the creation.
package com.example.books;

import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookCreateController {

    @PostMapping("/books")
    public BookDto create(@RequestBody BookDto draft) {
        return draft;
    }
}

record BookDto(Long id, String title) {
}
