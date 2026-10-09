// corpus: R3xx-04 vio page-vs-list
// lure: one endpoint answers Page<T>, its sibling a bare List — inconsistent by both shape and envelope.
package com.example.library;

import java.util.List;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LibraryController {

    @GetMapping("/books")
    public Page<BookDto> books(Pageable pageable) {
        return Page.empty();
    }

    @GetMapping("/authors")
    public List<AuthorDto> authors(Pageable pageable) {
        return List.of();
    }
}

record BookDto(Long id, String title) {
}

record AuthorDto(Long id, String penName) {
}
