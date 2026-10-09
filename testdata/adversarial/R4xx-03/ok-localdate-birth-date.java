// corpus: R4xx-03 ok localdate-birth-date
// lure: birthDate as LocalDate — a pure date with no time component; the pair stays silent.
package com.example.people;

import java.time.LocalDate;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class PersonController {

    @GetMapping("/people/{id}")
    public PersonDto get(@PathVariable Long id) {
        return null;
    }
}

record PersonDto(Long id, LocalDate birthDate) {
}
