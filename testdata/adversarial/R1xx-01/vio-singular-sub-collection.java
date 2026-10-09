// corpus: R1xx-01 vio singular-sub-collection
// lure: nested sub-collection segment stays singular between two path variables.
package com.example.library;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ReviewController {

    @GetMapping("/books/{bookId}/review/{reviewId}")
    public ReviewDto get(@PathVariable Long bookId, @PathVariable Long reviewId) {
        return null;
    }

    @GetMapping("/books/{bookId}/reviews")
    public List<ReviewDto> list(@PathVariable Long bookId) {
        return List.of();
    }
}

record ReviewDto(Long id, String comment) {
}
