// corpus: R1xx-03 ok single-level-collection
// lure: one collection level with its item variable — the minimal legal shape.
package com.example.catalog;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LibraryController {

    @GetMapping("/libraries/{libraryId}")
    public LibraryDto get(@PathVariable Long libraryId) {
        return null;
    }
}

record LibraryDto(Long id, String name) {
}
