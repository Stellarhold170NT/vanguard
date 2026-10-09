// corpus: R2xx-02 ok post-201-responsestatus
// lure: the same create with @ResponseStatus(CREATED) — 201 declared, both status rules silent.
package com.example.books;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookCreateController {

    @PostMapping("/books")
    @ResponseStatus(HttpStatus.CREATED)
    public BookDto create(@RequestBody BookDto draft) {
        return draft;
    }
}

record BookDto(Long id, String title) {
}
