// corpus: R1xx-02 ok noun-updates
// lure: "updates" is the plural noun (a feed of updates) — not the verb update.
package com.example.feed;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class FeedController {

    @GetMapping("/updates")
    public List<FeedItemDto> list() {
        return List.of();
    }
}

record FeedItemDto(Long id, String text) {
}
