package com.example.library.member;

import java.time.Instant;

/** The clean member DTO. */
public record MemberDto(Long id, Long libraryId, String fullName, Instant createdAt) {
}
