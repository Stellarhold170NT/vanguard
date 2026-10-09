// corpus: R3xx-02 vio array-response
// lure: an array return type — the oldest bare-collection spelling, still no envelope.
package com.example.tags;

import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class TagController {

    @GetMapping("/book-tags")
    public BookTagDto[] list(Pageable pageable) {
        return new BookTagDto[0];
    }
}

record BookTagDto(Long id, String slug) {
}
