// corpus: R1xx-05 vio self-id-mix
// lure: OrderDto names its own id orderId while MemberDto uses id — two self-id styles on one surface.
package com.example.shipments;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ShipmentController {

    @GetMapping("/orders/{id}")
    public OrderDto order(@PathVariable Long id) {
        return null;
    }

    @GetMapping("/members/{id}")
    public MemberDto member(@PathVariable Long id) {
        return null;
    }
}

record OrderDto(Long orderId, String tracking) {
}

record MemberDto(Long id, String name) {
}
