package com.example.library.order;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/**
 * Order API — demo-family showcase (w3-07): demoBadGetOrder fires the demo
 * trio R6xx-91 (marker name) + R6xx-92 (no response type) + R6xx-99
 * (trailing slash); archiveOrder isolates R6xx-92 (void endpoint).
 */
@RestController
@RequestMapping("/api/v1/orders")
public class OrderController {

    @GetMapping("/{orderId}/")
    public void demoBadGetOrder(@PathVariable Long orderId) {
    }

    @PostMapping("/{orderId}/archive")
    public void archiveOrder(@PathVariable Long orderId) {
    }
}
