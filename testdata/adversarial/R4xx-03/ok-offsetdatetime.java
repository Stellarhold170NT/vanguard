// corpus: R4xx-03 ok offsetdatetime
// lure: createdAt as OffsetDateTime — the zoned standard the rule wants.
package com.example.posts;

import java.time.OffsetDateTime;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class PostController {

    @GetMapping("/posts/{id}")
    public PostDto get(@PathVariable Long id) {
        return null;
    }
}

record PostDto(Long id, OffsetDateTime createdAt) {
}
