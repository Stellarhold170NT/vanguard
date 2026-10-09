// corpus: R1xx-04 ok numeric-segment
// lure: version segment /v2 is lower-case alphanumerics — legal kebab, nothing to normalize.
package com.example.versions;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class VersionedController {

    @GetMapping("/api/v2/books/{id}")
    public BookDto get(@PathVariable Long id) {
        return null;
    }
}

record BookDto(Long id, String title) {
}
