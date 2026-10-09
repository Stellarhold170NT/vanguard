package com.example.library.controller;

import com.example.library.dto.MemberDto;
import java.util.List;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1/members")
public class MemberController {

    @GetMapping("/{id}")
    public ResponseEntity<MemberDto> get(@PathVariable Long id) {
        return null;
    }

    @PostMapping
    public MemberDto register(@RequestBody MemberDto member) {
        return member;
    }

    @GetMapping
    public List<MemberDto> list() {
        return List.of();
    }
}
