// corpus: R3xx-01 ok list-limit-capped
// lure: a limit parameter with @Max — one query word, and both pagination rules stay silent.
package com.example.tags;

import org.springframework.data.domain.Page;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import jakarta.validation.constraints.Max;

@RestController
public class TagController {

    @GetMapping("/tags")
    public Page<TagDto> list(@RequestParam("limit") @Max(100) int limit) {
        return Page.empty();
    }
}

record TagDto(Long id, String label) {
}
