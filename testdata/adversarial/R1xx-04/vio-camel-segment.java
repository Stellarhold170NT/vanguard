// corpus: R1xx-04 vio camel-segment
// lure: camelCase literal segment /loanOrders — segments are kebab-case, not camel.
package com.example.loans;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LoanController {

    @GetMapping("/loanOrders")
    public String list() {
        return "[]";
    }
}
