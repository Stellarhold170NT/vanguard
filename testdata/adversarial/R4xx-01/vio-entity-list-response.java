// corpus: R4xx-01 vio entity-list-response
// lure: a bare List<Author> — the entity leak and the missing envelope at once.
package com.example.authors;

import java.util.List;

import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.Id;

@RestController
public class AuthorController {

    @GetMapping("/authors")
    public List<Author> list(Pageable pageable) {
        return List.of();
    }
}

@Entity
class Author {
    @Id
    @GeneratedValue
    private Long id;
    private String penName;

    public Long getId() {
        return id;
    }

    public String getPenName() {
        return penName;
    }
}
