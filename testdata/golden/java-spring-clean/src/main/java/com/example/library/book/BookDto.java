package com.example.library.book;

import java.time.Instant;

/** The clean book DTO — id/<resource>Id, lowerCamel fields, Instant times. */
public record BookDto(Long id, Long authorId, String title, Instant createdAt) {
}
