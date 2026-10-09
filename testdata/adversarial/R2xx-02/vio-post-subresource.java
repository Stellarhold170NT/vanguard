// corpus: R2xx-02 vio post-subresource
// lure: creating a sub-resource POST /books/{bookId}/reviews — creation hides one level down the path.
package com.example.reviews;

import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ReviewController {

    @PostMapping("/books/{bookId}/reviews")
    public ReviewDto create(@PathVariable Long bookId, @RequestBody ReviewDto draft) {
        return draft;
    }
}

record ReviewDto(Long id, String comment) {
}
