package com.example.library.member;

import jakarta.persistence.Entity;
import jakarta.persistence.Id;

@Entity
public class Member {

    @Id
    private Long id;

    private String name;

    public Long getId() {
        return id;
    }
}
