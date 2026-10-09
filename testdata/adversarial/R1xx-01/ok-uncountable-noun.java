// corpus: R1xx-01 ok uncountable-noun
// lure: mass noun /equipment has no plural form — flagging it would be a false positive.
package com.example.inventory;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class EquipmentController {

    @GetMapping("/equipment/{id}")
    public EquipmentDto get(@PathVariable Long id) {
        return null;
    }
}

record EquipmentDto(Long id, String kind) {
}
