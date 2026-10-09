// corpus: R6xx-01 vio unversioned-plain
// lure: the bare /orders base — versioning was never introduced on this controller.
package com.example.members;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/orders")
public class MemberOrderController {

    @GetMapping
    public OrderDto list() {
        return null;
    }
}

record OrderDto(Long id, String reference) {
}
