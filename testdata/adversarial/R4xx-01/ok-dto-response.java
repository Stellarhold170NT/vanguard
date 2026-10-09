// corpus: R4xx-01 ok dto-response
// lure: the entity exists in the file but the handler answers its DTO — the leak never happens.
package com.example.books;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.Id;

@RestController
public class BookController {

    @GetMapping("/books/{id}")
    public BookDto get(@PathVariable Long id) {
        return null;
    }
}

@Entity
class Book {
    @Id
    @GeneratedValue
    private Long id;
    private String title;

    public Long getId() {
        return id;
    }

    public String getTitle() {
        return title;
    }
}

record BookDto(Long id, String title) {
}
