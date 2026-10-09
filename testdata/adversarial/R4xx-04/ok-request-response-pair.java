// corpus: R4xx-04 ok request-response-pair
// lure: requests are *Request, responses are *Response — a consistent pairing convention.
package com.example.orders;

import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class OrderController {

    @PostMapping("/orders")
    public OrderResponse create(@RequestBody OrderRequest draft) {
        return null;
    }
}

record OrderRequest(String reference) {
}

record OrderResponse(Long id, String reference) {
}
