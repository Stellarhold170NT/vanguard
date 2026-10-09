// audit-sample: second JPA entity — gives R4xx-01's entity index a second
// sole entity so the sole-entity resolution is exercised honestly.
package com.youthunion.audit.entity;

import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;

import java.time.Instant;

@Entity
public class TrainingCourse {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    private String title;

    private Instant startsAt;

    public Long getId() {
        return id;
    }

    public String getTitle() {
        return title;
    }
}
