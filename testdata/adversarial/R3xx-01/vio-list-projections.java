// corpus: R3xx-01 vio list-projections
// lure: the rows are slim projections, yet the collection is still unbounded — slimness is not pagination.
package com.example.digests;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class DigestController {

    @GetMapping("/book-digests")
    public List<BookSummaryDto> list() {
        return List.of();
    }
}

record BookSummaryDto(Long id, String title) {
}
