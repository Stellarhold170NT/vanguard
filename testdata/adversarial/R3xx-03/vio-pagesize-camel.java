// corpus: R3xx-03 vio pagesize-camel
// lure: pageSize hides the size word in camelCase — the last-word read still catches it.
package com.example.members;

import org.springframework.data.domain.Page;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class MemberController {

    @GetMapping("/members")
    public Page<MemberDto> list(@RequestParam("pageSize") int pageSize) {
        return Page.empty();
    }
}

record MemberDto(Long id, String name) {
}
