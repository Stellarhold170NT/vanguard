// audit-sample: the verb-path controller (pattern w1-03 pain 1 —
// /update/draft, /update/submit, /search/all-by-condition beside the clean
// CRUD baseline; import leaks the entity through a generic argument).
package com.youthunion.audit.web.rest;

import java.util.List;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import com.youthunion.audit.dto.ImportRequest;
import com.youthunion.audit.dto.ImportResult;
import com.youthunion.audit.dto.MovementLogDto;
import com.youthunion.audit.entity.YouthMember;

@RestController
@RequestMapping("/api/youth-union/movement-logs")
public class MovementLogResource {

    @PostMapping("/update/draft")
    public void saveDraft(@RequestBody MovementLogDto draft) {
    }

    @PostMapping("/update/submit")
    public void submit(@RequestBody MovementLogDto draft) {
    }

    @GetMapping("/search/all-by-condition")
    public Page<MovementLogDto> search(Pageable pageable) {
        return Page.empty();
    }

    @GetMapping("/{id}")
    public MovementLogDto get(@org.springframework.web.bind.annotation.PathVariable Long id) {
        return new MovementLogDto(id, "summary", null);
    }

    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public MovementLogDto create(@RequestBody MovementLogDto dto) {
        return dto;
    }

    @PostMapping("/export")
    public byte[] export(Pageable pageable) {
        return new byte[0];
    }

    @PostMapping("/import")
    public ImportResult<YouthMember> importRows(@RequestBody ImportRequest request) {
        return new ImportResult<>(0, 0, List.of());
    }
}
