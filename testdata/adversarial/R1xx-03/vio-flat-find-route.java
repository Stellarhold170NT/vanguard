// corpus: R1xx-03 vio flat-find-route
// lure: /find-by-id/{id} replaces the resource hierarchy with a flat action route.
package com.example.inventory;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ItemController {

    @GetMapping("/find-by-id/{id}")
    public ItemDto get(@PathVariable Long id) {
        return null;
    }
}

record ItemDto(Long id, String sku) {
}
