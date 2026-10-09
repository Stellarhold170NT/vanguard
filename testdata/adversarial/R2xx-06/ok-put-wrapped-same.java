// corpus: R2xx-06 ok put-wrapped-same
// lure: the same full round-trip wrapped in ResponseEntity — unwrapped, payload equals response again.
package com.example.loans;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LoanPutController {

    @PutMapping("/loans/{id}")
    public ResponseEntity<LoanDto> put(@PathVariable Long id, @RequestBody LoanDto full) {
        return ResponseEntity.ok(full);
    }
}

record LoanDto(Long id, java.time.Instant dueAt) {
}
