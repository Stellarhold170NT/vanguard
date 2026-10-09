// corpus: R2xx-03 vio patch-full-dto
// lure: PATCH taking the same BookDto it returns — the full replacement belongs to PUT.
package com.example.books;

import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookPatchController {

    @PatchMapping("/books/{id}")
    public BookDto patch(@PathVariable Long id, @RequestBody BookDto full) {
        return full;
    }
}

record BookDto(Long id, String title) {
}
