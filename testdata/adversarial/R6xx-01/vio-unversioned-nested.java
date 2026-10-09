// corpus: R6xx-01 vio unversioned-nested
// lure: a nested base path /back-office still carries no /vN — depth is not versioning.
package com.example.admin;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/back-office")
public class BackOfficeController {

    @GetMapping("/orders")
    public OrderDto orders() {
        return null;
    }
}

record OrderDto(Long id, String reference) {
}
