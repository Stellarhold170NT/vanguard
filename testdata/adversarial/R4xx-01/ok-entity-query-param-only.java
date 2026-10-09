// corpus: R4xx-01 ok entity-query-param-only
// lure: the entity only shows up in a query binding — payload and response never touch it.
package com.example.catalog;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.Id;

@RestController
public class ItemController {

    @GetMapping("/items")
    public ItemDto get(@RequestParam Item probe) {
        return null;
    }
}

@Entity
class Item {
    @Id
    @GeneratedValue
    private Long id;
    private String sku;

    public Long getId() {
        return id;
    }

    public String getSku() {
        return sku;
    }
}

record ItemDto(Long id, String sku) {
}
