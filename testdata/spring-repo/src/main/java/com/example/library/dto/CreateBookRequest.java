package com.example.library.dto;

import jakarta.validation.constraints.NotBlank;

public class CreateBookRequest {

    @NotBlank
    private String title;

    private String author;

    public String getTitle() {
        return title;
    }

    public void setTitle(String title) {
        this.title = title;
    }

    public String getAuthor() {
        return author;
    }
}
