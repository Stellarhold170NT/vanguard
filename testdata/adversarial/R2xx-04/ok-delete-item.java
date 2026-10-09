// corpus: R2xx-04 ok delete-item
// lure: plain item DELETE with no body — the convention the app already follows.
package com.example.books;

import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookController {

    @DeleteMapping("/books/{id}")
    public void delete(@PathVariable Long id) {
    }
}
