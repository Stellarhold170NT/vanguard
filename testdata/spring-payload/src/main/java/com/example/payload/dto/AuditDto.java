package com.example.payload.dto;

import java.time.Instant;
import java.time.LocalDate;
import java.util.Date;

public record AuditDto(
        Long id,
        String created_by,
        Date updatedAt,
        LocalDate startDate,
        Instant recordedAt) {
}
