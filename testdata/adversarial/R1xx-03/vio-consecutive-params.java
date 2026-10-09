// corpus: R1xx-03 vio consecutive-params
// lure: two path variables in a row — the sub-collection that owns the second id is missing.
package com.example.orders;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class OrderController {

    @GetMapping("/orders/{orderId}/{itemId}")
    public ItemDto get(@PathVariable Long orderId, @PathVariable Long itemId) {
        return null;
    }
}

record ItemDto(Long id, String sku) {
}
