// corpus: R4xx-02 vio snake-case-field
// lure: field cover_image_url — snake_case inside a lowerCamel surface.
package com.example.books;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookController {

    @GetMapping("/books/{id}")
    public BookDto get(@PathVariable Long id) {
        return null;
    }
}

record BookDto(Long id, String title, String cover_image_url) {
}
