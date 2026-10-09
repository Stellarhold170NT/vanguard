// corpus: R3xx-03 vio limit-unbounded
// lure: the limit spelling of the same trap — last-word matching makes limit a page size too.
package com.example.loans;

import org.springframework.data.domain.Page;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LoanController {

    @GetMapping("/loans")
    public Page<LoanDto> list(@RequestParam("limit") int limit) {
        return Page.empty();
    }
}

record LoanDto(Long id, java.time.Instant dueAt) {
}
