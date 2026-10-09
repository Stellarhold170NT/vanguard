package com.example.library.dto;

import jakarta.validation.constraints.Size;

public record MemberDto(
        Long id,
        @Size(max = 100) String name,
        String email) {
}
