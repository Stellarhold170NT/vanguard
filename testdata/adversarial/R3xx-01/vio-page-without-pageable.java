// corpus: R3xx-01 vio page-without-pageable
// lure: the response is a Page but the handler takes no Pageable — the envelope decorates an unbounded query.
package com.example.orders;

import org.springframework.data.domain.Page;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class OrderController {

    @GetMapping("/orders")
    public Page<OrderDto> list() {
        return Page.empty();
    }
}

record OrderDto(Long id, String reference) {
}
