package com.example.library.domain;

import jakarta.persistence.Entity;
import jakarta.persistence.Id;

/** The ORM entity — must never reach the wire (R4xx-01 showcase). */
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
