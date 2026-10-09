package com.example.library.member;

import org.springframework.data.domain.Page;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

/**
 * The clean member API (w3-07 golden showcase, clean case): no GET bodies,
 * no DELETE bodies, typed responses everywhere, dedicated PATCH payload,
 * Pageable list — R2xx-01..06 and R6xx-92/93 stay silent.
 */
@RestController
@RequestMapping("/api/v1/members")
public class MemberController {

    @GetMapping("/auto-complete")
    public MemberSuggestions autoComplete(@RequestParam String term) {
        return null;
    }

    @GetMapping
    public Page<MemberDto> listMembers(org.springframework.data.domain.Pageable pageable) {
        return null;
    }

    @GetMapping("/{memberId}")
    public MemberDto getMember(@PathVariable Long memberId) {
        return null;
    }

    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public MemberDto createMember(@RequestBody MemberDto dto) {
        return null;
    }

    @PatchMapping("/{memberId}")
    public MemberDto patchMember(@RequestBody MemberPatch patch, @PathVariable Long memberId) {
        return null;
    }

    @DeleteMapping("/{memberId}")
    public MemberDto deleteMember(@PathVariable Long memberId) {
        return null;
    }
}
