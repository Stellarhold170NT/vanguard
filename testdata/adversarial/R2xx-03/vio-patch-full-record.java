// corpus: R2xx-03 vio patch-full-record
// lure: PATCH round-tripping a Java record — same type in, same type out, still a full replacement.
package com.example.copies;

import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class CopyPatchController {

    @PatchMapping("/copies/{id}")
    public CopyRecord patch(@PathVariable Long id, @RequestBody CopyRecord full) {
        return full;
    }
}

record CopyRecord(Long id, String shelf, boolean lent) {
}
