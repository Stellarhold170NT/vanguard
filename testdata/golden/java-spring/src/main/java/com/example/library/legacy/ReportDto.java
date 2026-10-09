package com.example.library.legacy;

import java.time.Instant;

/** The clean report DTO. */
public record ReportDto(Long id, String title, Instant generatedAt) {
}
