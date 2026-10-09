// corpus: R1xx-05 vio mixed-json-key
// lure: JSON keys split conventions — user_id snake against orderId camel in one DTO.
package com.example.orders;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import com.fasterxml.jackson.annotation.JsonProperty;

@RestController
public class FulfillmentController {

    @GetMapping("/fulfillments/{id}")
    public FulfillmentDto get(@PathVariable Long id) {
        return null;
    }
}

record FulfillmentDto(
        @JsonProperty("user_id") Long userId,
        @JsonProperty("orderId") Long orderId) {
}
