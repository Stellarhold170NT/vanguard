package com.example.library.member;

import java.time.Instant;
import java.util.List;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;

/**
 * Member API. Violations on purpose (w3-07 golden showcase — mapped in
 * README.md):
 *
 * <ul>
 *   <li>autoComplete — R2xx-01 ERROR (GET with a request body) + demo
 *       R6xx-93 INFO; the envelope return keeps the pagination family
 *       silent (the clean counterpart for R3xx-01/02)</li>
 *   <li>deleteMember — R2xx-04 (DELETE with a body)</li>
 *   <li>updateMember — R2xx-06 (PUT payload type differs from the response
 *       type; the clean counterpart is BookController-style same-type PUT
 *       in the clean case)</li>
 *   <li>patchMember — R2xx-03 (PATCH body typed as the full entity)</li>
 *   <li>exportMembers — R2xx-05 (action path on GET) + R3xx-01
 *       (unpaginated collection) + R3xx-02 (bare List)</li>
 * </ul>
 */
@RestController
@RequestMapping("/api/v1/members")
public class MemberController {

    @GetMapping("/auto-complete")
    public MemberSuggestions autoComplete(@RequestBody MemberQuery query) {
        return null;
    }

    @DeleteMapping("/{memberId}")
    public MemberDto deleteMember(@RequestBody List<Long> memberIds) {
        return null;
    }

    @PutMapping("/{memberId}")
    public MemberDto updateMember(@RequestBody MemberUpdateRequest request, @PathVariable Long memberId) {
        return null;
    }

    @PatchMapping("/{memberId}")
    public MemberDto patchMember(@RequestBody MemberDto fullEntity, @PathVariable Long memberId) {
        return null;
    }

    @GetMapping("/export")
    public List<MemberDto> exportMembers() {
        return null;
    }
}
