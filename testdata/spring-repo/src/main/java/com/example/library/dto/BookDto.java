package com.example.library.dto;

import com.fasterxml.jackson.annotation.JsonProperty;

public record BookDto(
        @JsonProperty("book_id") Long id,
        String title,
        String author) {
}
