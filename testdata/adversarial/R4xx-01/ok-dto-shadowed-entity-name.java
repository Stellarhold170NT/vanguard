// corpus: R4xx-01 ok dto-shadowed-entity-name
// lure: a DTO shares the entity's simple name — first-declaration-wins makes the reference ambiguous, the rule stays silent.
package com.example.members;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.Id;

@RestController
public class MemberController {

    @GetMapping("/members/{id}")
    public Member get(@PathVariable Long id) {
        return null;
    }
}

record Member(Long id, String name) {
}

@Entity
class Member {
    @Id
    @GeneratedValue
    private Long id;
    private String name;

    public Long getId() {
        return id;
    }

    public String getName() {
        return name;
    }
}
