// corpus: R3xx-01 ok list-page-size-params
// lure: explicit page/size query parameters — the framework-free pagination shape.
package com.example.members;

import org.springframework.data.domain.Page;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class MemberController {

    @GetMapping("/members")
    public Page<MemberDto> list(@RequestParam int page, @RequestParam int size) {
        return Page.empty();
    }
}

record MemberDto(Long id, String name) {
}
