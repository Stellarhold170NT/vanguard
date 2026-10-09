// corpus: _global vio nested-generic-wildcard
// lure: PageResponse<List<? extends Youth>> — nested generics with a bounded wildcard around the entity.
package com.example.youth;

import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.Id;

@RestController
public class ProjectionController {

    @GetMapping("/youth-projections")
    public PageResponse<List<? extends Youth>> list(Pageable pageable) {
        return null;
    }
}

record PageResponse<T>(java.util.List<T> items, long total) {
}

@Entity
class Youth {
    @Id
    @GeneratedValue
    private Long id;
    private String cohort;

    public Long getId() {
        return id;
    }

    public String getCohort() {
        return cohort;
    }
}
