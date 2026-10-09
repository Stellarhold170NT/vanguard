// corpus: R1xx-05 ok self-and-reference
// lure: the charter convention itself — id for the self id, <resource>Id for references.
package com.example.loans;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LoanController {

    @GetMapping("/loans/{id}")
    public LoanDto get(@PathVariable Long id) {
        return null;
    }
}

record LoanDto(Long id, Long bookId, Long memberId) {
}
