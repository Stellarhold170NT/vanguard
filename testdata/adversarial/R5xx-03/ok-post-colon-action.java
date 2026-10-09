// corpus: R5xx-03 ok post-colon-action
// lure: POST /books/{bookId}:restore — the colon custom-method shape is not a create route, so no status demand.
package com.example.books;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class RestoreController {

    @PostMapping("/books/{bookId}:restore")
    public ResponseEntity<Void> restore(@PathVariable Long bookId) {
        return ResponseEntity.accepted().build();
    }
}
