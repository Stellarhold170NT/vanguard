// corpus: R2xx-06 vio put-partial-name
// lure: the method is even named partialUpdate* — the PUT verb still owns full replacements.
package com.example.members;

import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class MemberController {

    @PutMapping("/members/{id}")
    public MemberDto partialUpdateBook(@PathVariable Long id,
            @RequestBody MemberPatchRequest patch) {
        return null;
    }
}

record MemberPatchRequest(String nickname) {
}

record MemberDto(Long id, String name) {
}
