// corpus: R2xx-03 vio patch-full-entity
// lure: PATCH round-tripping the ORM entity — a full replacement that also leaks the entity to the wire.
package com.example.domain;

import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class MemberPatchController {

    @PatchMapping("/members/{id}")
    public Member patch(@PathVariable Long id, @RequestBody Member full) {
        return full;
    }
}

record Member(Long id, String name, String email) {
}
