// corpus: R2xx-06 vio put-entity-response
// lure: PUT whose response is the ORM entity while the payload is the update DTO — partial shape plus a wire leak.
package com.example.catalog;

import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ItemController {

    @PutMapping("/items/{id}")
    public Item put(@PathVariable Long id, @RequestBody ItemUpdateRequest patch) {
        return null;
    }
}

record ItemUpdateRequest(String sku) {
}

record Item(Long id, String sku, int stock) {
}
