// corpus: R2xx-06 ok put-full-same
// lure: PUT round-tripping the same BookDto — payload equals response, the full replacement it should be.
package com.example.books;

import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookPutController {

    @PutMapping("/books/{id}")
    public BookDto put(@PathVariable Long id, @RequestBody BookDto full) {
        return full;
    }
}

record BookDto(Long id, String title) {
}
