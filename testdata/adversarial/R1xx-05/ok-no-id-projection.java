// corpus: R1xx-05 ok no-id-projection
// lure: a projection DTO without any id-shaped field — neither id check has an opinion here.
package com.example.digest;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class DigestController {

    @GetMapping("/digests/{id}")
    public BookDigestDto get(@PathVariable Long id) {
        return null;
    }
}

record BookDigestDto(String title, String authorName, int pageCount) {
}
