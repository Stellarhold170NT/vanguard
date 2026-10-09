// corpus: R4xx-04 vio response-vs-dto
// lure: LoanResponse next to BookDto — the *Response and *DTO conventions share one controller.
package com.example.loans;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LoanController {

    @GetMapping("/loans/{id}")
    public LoanResponse get(@PathVariable Long id) {
        return null;
    }

    @GetMapping("/loans/{id}/book")
    public BookDto getBook(@PathVariable Long id) {
        return null;
    }
}

record LoanResponse(Long id, java.time.Instant dueAt) {
}

record BookDto(Long id, String title) {
}
