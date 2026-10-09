// corpus: R6xx-01 ok versioned-base
// lure: /api/v1/orders — the version segment the pattern wants.
package com.example.orders;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/v1/orders")
public class OrderController {

    @GetMapping
    public OrderDto list() {
        return null;
    }
}

record OrderDto(Long id, String reference) {
}
