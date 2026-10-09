// corpus: R4xx-02 ok json-camel-key
// lure: an explicit @JsonProperty that is still lowerCamel — the override changes nothing the rule measures.
package com.example.orders;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import com.fasterxml.jackson.annotation.JsonProperty;

@RestController
public class OrderController {

    @GetMapping("/orders/{id}")
    public OrderDto get(@PathVariable Long id) {
        return null;
    }
}

record OrderDto(Long id, @JsonProperty("orderReference") String reference) {
}
