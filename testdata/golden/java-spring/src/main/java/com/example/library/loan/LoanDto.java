package com.example.library.loan;

import java.time.Instant;

/** The clean loan DTO. */
public record LoanDto(Long id, Long memberId, Instant borrowedAt) {
}
