// corpus: R1xx-02 vio create-token
// lure: POST /create/books — the create verb leads the route instead of the collection.
package com.example.copies;

import org.springframework.data.domain.Page;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class CopyController {

    @PostMapping("/create/books")
    public Page<BookCopyDto> create() {
        return Page.empty();
    }
}

record BookCopyDto(Long id, String isbn) {
}
