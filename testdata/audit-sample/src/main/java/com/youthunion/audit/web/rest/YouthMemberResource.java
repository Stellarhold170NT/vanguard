// audit-sample: the pain-7 epicenter controller (pattern w1-03 pain 7 —
// YouthResource): GET with body, DELETE with body, entity on the wire,
// PUT partial update — beside the clean JHipster CRUD baseline.
package com.youthunion.audit.web.rest;

import java.util.List;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import com.youthunion.audit.dto.YouthAutocompleteRequest;
import com.youthunion.audit.dto.YouthMemberDto;
import com.youthunion.audit.dto.YouthMemberPatch;
import com.youthunion.audit.dto.YouthStatusUpdate;
import com.youthunion.audit.entity.YouthMember;

@RestController
@RequestMapping("/api/youth-union/youths")
public class YouthMemberResource {

    @GetMapping("/{id}")
    public YouthMemberDto get(@org.springframework.web.bind.annotation.PathVariable Long id) {
        return new YouthMemberDto(id, "Full Name", "TC1", null);
    }

    @GetMapping
    public Page<YouthMemberDto> list(Pageable pageable) {
        return Page.empty();
    }

    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public YouthMemberDto create(@RequestBody YouthMemberDto dto) {
        return dto;
    }

    @PutMapping("/{id}")
    public YouthMemberDto replace(@RequestBody YouthMemberDto dto) {
        return dto;
    }

    @PatchMapping("/{id}")
    public YouthMemberDto patch(@RequestBody YouthMemberPatch patch) {
        return new YouthMemberDto(1L, patch.fullName(), "TC1", null);
    }

    @DeleteMapping("/{id}")
    public void remove(@org.springframework.web.bind.annotation.PathVariable Long id) {
    }

    @GetMapping("/auto-complete")
    public List<String> autoComplete(@RequestBody YouthAutocompleteRequest request) {
        return List.of();
    }

    @DeleteMapping
    public void removeBatch(@RequestBody List<String> ids) {
    }

    @GetMapping("/current")
    public YouthMember current() {
        return new YouthMember();
    }

    @PutMapping("/{id}/status")
    public YouthMemberDto updateStatus(@RequestBody YouthStatusUpdate update) {
        return new YouthMemberDto(1L, "Full Name", update.status(), null);
    }
}
