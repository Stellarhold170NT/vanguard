// corpus: R5xx-03 ok post-201-created
// lure: the same create answering 201 Created — the status semantics the rule wants.
package com.example.copies;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class CopyController {

    @PostMapping("/book-copies")
    @ResponseStatus(HttpStatus.CREATED)
    public BookCopyDto create(@RequestBody BookCopyDto draft) {
        return draft;
    }
}

record BookCopyDto(Long id, String isbn) {
}
