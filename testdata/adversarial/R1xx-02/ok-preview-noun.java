// corpus: R1xx-02 ok preview-noun
// lure: plural noun /previews sits one 's' from the action preview — token equality keeps it silent.
package com.example.gallery;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class PreviewController {

    @GetMapping("/previews/{id}")
    public PreviewDto get(@PathVariable Long id) {
        return null;
    }
}

record PreviewDto(Long id, String image) {
}
