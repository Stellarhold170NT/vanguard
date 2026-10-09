// corpus: R1xx-03 ok nested-hierarchy
// lure: textbook /collection/{id}/sub-collection/{subId} shape — every variable sits under its resource.
package com.example.library;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class AuthorBookController {

    @GetMapping("/authors/{authorId}/books/{bookId}")
    public BookDto get(@PathVariable Long authorId, @PathVariable Long bookId) {
        return null;
    }
}

record BookDto(Long id, String title) {
}
