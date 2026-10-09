// corpus: R2xx-03 ok patch-void-response
// lure: PATCH with no declared response — without one the full-vs-partial heuristic stays silent.
package com.example.tags;

import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class TagPatchController {

    @PatchMapping("/tags/{id}")
    public void patch(@PathVariable Long id, @RequestBody TagUpdateRequest patch) {
    }
}

record TagUpdateRequest(String label) {
}
