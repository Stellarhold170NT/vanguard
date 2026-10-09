// corpus: R4xx-03 vio instant-as-string
// lure: publishedAt typed String — a timestamp serialized as free text.
package com.example.posts;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class PostController {

    @GetMapping("/posts/{id}")
    public PostDto get(@PathVariable Long id) {
        return null;
    }
}

record PostDto(Long id, String publishedAt) {
}
