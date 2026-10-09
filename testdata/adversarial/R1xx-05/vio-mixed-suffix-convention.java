// corpus: R1xx-05 vio mixed-suffix-convention
// lure: one DTO mixes bookID, member_id and publisherId — three spellings of the same convention.
package com.example.inventory;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class StockController {

    @GetMapping("/items/{id}")
    public StockItemDto get(@PathVariable Long id) {
        return null;
    }
}

record StockItemDto(Long bookID, Long member_id, Long publisherId, String title) {
}
