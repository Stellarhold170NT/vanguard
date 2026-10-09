// corpus: R2xx-03 ok patch-map-partial
// lure: PATCH with Map<String,Object> — a field bag, not the entity; the heuristic cannot call it full.
package com.example.members;

import java.util.Map;

import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class MemberPatchController {

    @PatchMapping("/members/{id}")
    public MemberDto patch(@PathVariable Long id, @RequestBody Map<String, Object> fields) {
        return null;
    }
}

record MemberDto(Long id, String name) {
}
