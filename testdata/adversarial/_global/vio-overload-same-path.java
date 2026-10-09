// corpus: _global vio overload-same-path
// lure: PATCH and GET overloads share /books/{id} — exactly one R2xx-03 finding, no doubling.
package com.example.books;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookController {

    @PatchMapping("/books/{id}")
    public BookDto patch(@PathVariable Long id, @RequestBody BookDto full) {
        return full;
    }

    @GetMapping("/books/{id}")
    public BookDto get(@PathVariable Long id) {
        return null;
    }
}

record BookDto(Long id, String title) {
}
