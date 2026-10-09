package com.example.library.domain;

import jakarta.persistence.Entity;
import jakarta.persistence.Id;

/**
 * The ORM entity EXISTS in this repo but is never referenced by a payload
 * or response — the clean counterpart of R4xx-01 (w3-07 golden showcase).
 */
@Entity
public class Book {

    @Id
    private Long id;

    private String title;

    private boolean available;

    public Long getId() {
        return id;
    }

    public String getTitle() {
        return title;
    }

    public boolean isAvailable() {
        return available;
    }
}
