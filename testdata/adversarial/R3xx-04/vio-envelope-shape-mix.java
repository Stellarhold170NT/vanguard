// corpus: R3xx-04 vio envelope-shape-mix
// lure: a Page envelope on one endpoint, an app-owned history record on the next — inconsistent shapes.
package com.example.orders;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class OrderController {

    @GetMapping("/orders")
    public Page<OrderDto> list(Pageable pageable) {
        return Page.empty();
    }

    @GetMapping("/orders/history")
    public OrderHistoryPage history(Pageable pageable) {
        return new OrderHistoryPage(java.util.List.of(), 0);
    }
}

record OrderHistoryPage(java.util.List<OrderDto> items, long total) {
}

record OrderDto(Long id, String reference) {
}
