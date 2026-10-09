// corpus: R2xx-05 ok post-colon-restore
// lure: AIP-136 custom method POST /books/{bookId}:restore — POST already satisfies the convention.
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
