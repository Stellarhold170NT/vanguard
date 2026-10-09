// corpus: R1xx-01 ok camel-plural
// lure: camelCase plural /bookShelves reads plural at the token level — the casing split is R1xx-04's subject, not this rule's.
package com.example.shelves;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ShelfController {

    @GetMapping("/bookShelves/{id}")
    public BookShelfDto get(@PathVariable Long id) {
        return null;
    }
}

record BookShelfDto(Long id, String label) {
}
