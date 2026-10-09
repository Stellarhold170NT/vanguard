// corpus: R2xx-03 ok patch-partial-dto
// lure: PATCH with a dedicated partial DTO — response differs from payload, the rule stays out.
package com.example.books;

import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookPatchController {

    @PatchMapping("/books/{id}")
    public BookDto patch(@PathVariable Long id, @RequestBody BookUpdateRequest patch) {
        return null;
    }
}

record BookUpdateRequest(String title) {
}

record BookDto(Long id, String title) {
}
