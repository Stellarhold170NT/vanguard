// corpus: R1xx-02 vio fetch-under-resource
// lure: POST /books/fetch — the action verb hides under the collection; only the path rule may flag it (R2xx-05 stays silent on POST).
package com.example.books;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class FetchController {

    @PostMapping("/books/fetch")
    public ResponseEntity<Void> fetch() {
        return ResponseEntity.accepted().build();
    }
}
