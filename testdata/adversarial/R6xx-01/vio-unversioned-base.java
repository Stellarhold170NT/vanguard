// corpus: R6xx-01 vio unversioned-base
// lure: base path /api/orders with no version segment — clients pin to an unversioned contract.
package com.example.orders;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/orders")
public class OrderController {

    @GetMapping
    public OrderDto get() {
        return null;
    }
}

record OrderDto(Long id, String reference) {
}
