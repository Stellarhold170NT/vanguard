// corpus: R1xx-02 vio camel-verb-path
// lure: RPC-style camelCase path /getUserById — the verb hides inside one segment.
package com.example.members;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class MemberController {

    @GetMapping("/getUserById")
    public MemberDto get() {
        return null;
    }
}

record MemberDto(Long id, String name) {
}
