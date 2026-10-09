// audit-sample: the casing controller (pattern w1-03 pain 2 —
// /search/all-by-Name, uppercase inside a kebab segment) plus the full
// PATCH that R2xx-03's equal-type heuristic reads.
package com.youthunion.audit.web.rest;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import com.youthunion.audit.dto.CommentDto;
import com.youthunion.audit.dto.MagazinePostDto;

@RestController
@RequestMapping("/api/youth-union/magazine-posts")
public class MagazinePostResource {

    @GetMapping("/search/all-by-Name")
    public List<MagazinePostDto> searchByName() {
        return List.of();
    }

    @GetMapping("/search/all-by-condition")
    public List<MagazinePostDto> searchByCondition() {
        return List.of();
    }

    @GetMapping("/{magazinePostId}")
    public MagazinePostDto get(@org.springframework.web.bind.annotation.PathVariable Long magazinePostId) {
        return new MagazinePostDto(magazinePostId, "post");
    }

    @GetMapping("/{magazineId}/comments")
    public List<CommentDto> comments(@org.springframework.web.bind.annotation.PathVariable Long magazineId) {
        return List.of();
    }

    @PatchMapping("/{magazinePostId}")
    public MagazinePostDto patch(@RequestBody MagazinePostDto dto) {
        return dto;
    }
}
