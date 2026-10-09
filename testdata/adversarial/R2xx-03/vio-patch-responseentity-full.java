// corpus: R2xx-03 vio patch-responseentity-full
// lure: the ResponseEntity wrapper hides the full round-trip — unwrapped, payload still equals the response.
package com.example.loans;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LoanPatchController {

    @PatchMapping("/loans/{id}")
    public ResponseEntity<LoanDto> patch(@PathVariable Long id, @RequestBody LoanDto full) {
        return ResponseEntity.ok(full);
    }
}

record LoanDto(Long id, java.time.Instant dueAt) {
}
