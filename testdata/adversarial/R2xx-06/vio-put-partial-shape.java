// corpus: R2xx-06 vio put-partial-shape
// lure: PUT taking BookUpdateRequest while answering BookDto — the mismatch reads as a partial update.
package com.example.books;

import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookPutController {

    @PutMapping("/books/{id}")
    public BookDto put(@PathVariable Long id, @RequestBody BookUpdateRequest patch) {
        return null;
    }
}

record BookUpdateRequest(String title) {
}

record BookDto(Long id, String title) {
}
