// corpus: R1xx-05 ok consistent-reference-ids
// lure: every reference id uses the same <resource>Id spelling — one convention, nothing to report.
package com.example.catalog;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class AuthorController {

    @GetMapping("/authors/{id}")
    public AuthorSummaryDto get(@PathVariable Long id) {
        return null;
    }
}

record AuthorSummaryDto(Long authorId, Long publisherId, String penName) {
}
