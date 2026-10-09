package com.example.library.author;

import java.time.Instant;

/** The clean author DTO. */
public record AuthorDto(Long id, String name, Instant createdAt) {
}
