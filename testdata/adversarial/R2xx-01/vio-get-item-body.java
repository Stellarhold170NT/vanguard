// corpus: R2xx-01 vio get-item-body
// lure: GET on an item path with a body — /books/{bookId}/content reads a range it should take as query parameters.
package com.example.content;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ContentController {

    @GetMapping("/books/{bookId}/content")
    public ContentDto read(@PathVariable Long bookId, @RequestBody ContentRange range) {
        return null;
    }
}

record ContentRange(int fromPage, int toPage) {
}

record ContentDto(Long bookId, String chapter) {
}
