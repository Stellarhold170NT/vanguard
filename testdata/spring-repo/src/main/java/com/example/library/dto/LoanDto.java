package com.example.library.dto;

import com.fasterxml.jackson.annotation.JsonProperty;

public record LoanDto(
        @JsonProperty("loan_id") Long id,
        Long bookId,
        Long memberId) {
}
