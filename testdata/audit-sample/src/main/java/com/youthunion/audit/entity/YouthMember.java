// audit-sample: JPA entity exposed on the wire (pattern w1-03 pain 6 —
// YouthResource GET /current returns the entity). Fictional youth-union
// domain; no military-youth source copied (charter §12).
package com.youthunion.audit.entity;

import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;

import java.time.Instant;

@Entity
@Table(name = "youth_member")
public class YouthMember {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    private String fullName;

    private String unitCode;

    private Instant createdAt;

    public Long getId() {
        return id;
    }

    public String getFullName() {
        return fullName;
    }

    public Instant getCreatedAt() {
        return createdAt;
    }
}
