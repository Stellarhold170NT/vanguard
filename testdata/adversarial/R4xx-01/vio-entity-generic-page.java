// corpus: R4xx-01 vio entity-generic-page
// lure: the entity hides one generic deep inside Page<Author> — the read must look through the wrapper.
package com.example.authors;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.Id;

@RestController
public class AuthorController {

    @GetMapping("/authors")
    public Page<Author> list(Pageable pageable) {
        return Page.empty();
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
