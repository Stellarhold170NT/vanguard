// audit-sample: the second pagination shape (pattern w1-03 pain 4 — the
// same app speaking Spring Page here and PageResponse in ReportResource),
// plus the /by-member qualifier route.
package com.youthunion.audit.web.rest;

import java.util.List;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import com.youthunion.audit.dto.UnionFeeDto;

@RestController
@RequestMapping("/api/youth-union/union-fees")
public class UnionFeeResource {

    @GetMapping
    public Page<UnionFeeDto> list(Pageable pageable) {
        return Page.empty();
    }

    @GetMapping("/{id}")
    public UnionFeeDto get(@org.springframework.web.bind.annotation.PathVariable Long id) {
        return new UnionFeeDto(id, "Q1-2026", 0);
    }

    @GetMapping("/by-member/{memberId}")
    public List<UnionFeeDto> byMember(@org.springframework.web.bind.annotation.PathVariable Long memberId) {
        return List.of();
    }
}
