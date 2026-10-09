// corpus: R4xx-02 vio pascal-field
// lure: field FirstName — PascalCase leak from a copy-paste.
package com.example.staff;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class StaffController {

    @GetMapping("/staff/{id}")
    public StaffDto get(@PathVariable Long id) {
        return null;
    }
}

record StaffDto(Long id, String FirstName, String lastName) {
}
