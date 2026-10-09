// corpus: R1xx-04 ok kebab-standard
// lure: plain kebab segments with a camelCase variable — the convention the app already follows.
package com.example.audiobooks;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class AudioBookController {

    @GetMapping("/audio-books/{bookId}")
    public AudioBookDto get(@PathVariable Long bookId) {
        return null;
    }
}

record AudioBookDto(Long id, String narrator) {
}
