// audit-sample: the flat-route controller (pattern w1-03 pain 1+3 —
// /find-by-id/{id}, /find-all-id-active, @DeleteMapping("/delete") beside
// the clean CRUD baseline; the create POST misses 201).
package com.youthunion.audit.web.rest;

import java.util.List;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import com.youthunion.audit.dto.TrainingCourseDto;

@RestController
@RequestMapping("/api/youth-union/training-courses")
public class TrainingCourseResource {

    @GetMapping("/find-by-id/{id}")
    public TrainingCourseDto findById(@org.springframework.web.bind.annotation.PathVariable Long id) {
        return new TrainingCourseDto(id, "course");
    }

    @GetMapping("/find-all-id-active")
    public List<Long> findAllIdActive() {
        return List.of();
    }

    @DeleteMapping("/delete")
    public void remove(@org.springframework.web.bind.annotation.PathVariable Long id) {
    }

    @GetMapping("/{id}")
    public TrainingCourseDto get(@org.springframework.web.bind.annotation.PathVariable Long id) {
        return new TrainingCourseDto(id, "course");
    }

    @GetMapping
    public Page<TrainingCourseDto> list(Pageable pageable) {
        return Page.empty();
    }

    @PostMapping
    public TrainingCourseDto create(@RequestBody TrainingCourseDto dto) {
        return dto;
    }
}
