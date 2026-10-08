package com.example.books;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import jakarta.validation.constraints.Size;

@JsonInclude(JsonInclude.Include.NON_NULL)
public record BookDto(
        @JsonProperty("book_id") Long id,
        String title,
        @Size(max = 200) String summary) {
}
