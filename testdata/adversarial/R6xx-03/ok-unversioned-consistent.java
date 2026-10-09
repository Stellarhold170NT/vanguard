// corpus: R6xx-03 ok unversioned-consistent
// lure: no version segments anywhere — consistent, and R6xx-01 stays default-off in this pass.
package com.example.members;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class MemberController {

    @GetMapping("/api/members")
    public Page<MemberDto> members(Pageable pageable) {
        return Page.empty();
    }

    @GetMapping("/api/trainers")
    public Page<TrainerDto> trainers(Pageable pageable) {
        return Page.empty();
    }
}

record MemberDto(Long id, String name) {
}

record TrainerDto(Long id, String specialty) {
}
