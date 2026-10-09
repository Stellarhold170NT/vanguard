package com.example.payload.domain;

import jakarta.persistence.Entity;
import jakarta.persistence.Id;

@Entity
public class Customer {

    @Id
    private Long id;

    private String full_name;

    public Long getId() {
        return id;
    }

    public String getFullName() {
        return full_name;
    }
}
