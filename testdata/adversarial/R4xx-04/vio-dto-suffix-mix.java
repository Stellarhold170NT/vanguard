// corpus: R4xx-04 vio dto-suffix-mix
// lure: responses are *DTO and requests are *Request — two suffix conventions in one module.
package com.example.library;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LibraryController {

    @GetMapping("/books/{id}")
    public BookDto get(@PathVariable Long id) {
        return null;
    }

    @PostMapping("/books")
    public void create(@RequestBody BookRequest draft) {
    }
}

record BookDto(Long id, String title) {
}

record BookRequest(String title) {
}
