// corpus: R1xx-02 vio update-draft-path
// lure: two-stage action phrase /update/draft — the verb hides under a sub-path.
package com.example.workflow;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class DraftController {

    @PostMapping("/update/draft")
    public ResponseEntity<Void> update(@PathVariable Long id) {
        return ResponseEntity.ok().build();
    }
}
