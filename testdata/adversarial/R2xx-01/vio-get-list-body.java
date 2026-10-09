// corpus: R2xx-01 vio get-list-body
// lure: GET whose body is a raw id list — the bulk-read shape that silently fails on real clients.
package com.example.bulk;

import java.util.List;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BulkReadController {

    @GetMapping("/books/batch")
    public Page<BookDto> byIds(@RequestBody List<Long> ids, Pageable pageable) {
        return Page.empty();
    }
}

record BookDto(Long id, String title) {
}
