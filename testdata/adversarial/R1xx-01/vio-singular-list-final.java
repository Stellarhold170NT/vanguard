// corpus: R1xx-01 vio singular-list-final
// lure: collection GET whose final noun is one token from the plural /books — but paginated, so only the plural rule may speak.
package com.example.catalog;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ShelfViewController {

    @GetMapping("/book")
    public Page<BookShelfDto> list(Pageable pageable) {
        return Page.empty();
    }
}

record BookShelfDto(Long id, String label) {
}
