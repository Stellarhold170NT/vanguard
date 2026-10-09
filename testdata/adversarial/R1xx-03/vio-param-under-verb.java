// corpus: R1xx-03 vio param-under-verb
// lure: {memberId} sits under the action segment /update — the item belongs to the resource, not the action.
package com.example.members;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class MemberAdminController {

    @GetMapping("/members/update/{memberId}")
    public MemberDto get(@PathVariable Long memberId) {
        return null;
    }
}

record MemberDto(Long id, String name) {
}
