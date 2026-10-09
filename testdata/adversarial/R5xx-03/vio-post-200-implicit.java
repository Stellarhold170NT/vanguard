// corpus: R5xx-03 vio post-200-implicit
// lure: a create-shaped POST answering the implicit 200 — the created resource is not 201.
package com.example.copies;

import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class CopyController {

    @PostMapping("/book-copies")
    public BookCopyDto create(@RequestBody BookCopyDto draft) {
        return draft;
    }
}

record BookCopyDto(Long id, String isbn) {
}
