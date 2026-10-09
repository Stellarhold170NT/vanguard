// corpus: R2xx-04 ok delete-query-ids
// lure: bulk delete via repeated query parameters — the documented legal shape.
package com.example.books;

import java.util.List;

import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BookPurgeController {

    @DeleteMapping("/books")
    public void purge(@RequestParam("ids") List<Long> ids) {
    }
}
