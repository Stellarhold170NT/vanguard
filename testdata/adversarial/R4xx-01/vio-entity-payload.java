// corpus: R4xx-01 vio entity-payload
// lure: POST taking the entity as the request body — client-controlled columns, mass assignment by construction.
package com.example.books;

import org.springframework.data.domain.Page;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.Id;

@RestController
public class BookCreateController {

    @PostMapping("/books")
    public Page<BookDto> create(@RequestBody Book draft) {
        return Page.empty();
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
