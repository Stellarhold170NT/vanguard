// corpus: R1xx-02 ok noun-getter
// lure: "getter" is a domain noun, not the verb get — only exact tokens are verbs.
package com.example.billing;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BudgetController {

    @GetMapping("/budget-getter/{id}")
    public BudgetDto get(@PathVariable Long id) {
        return null;
    }
}

record BudgetDto(Long id, Long amountCents) {
}
