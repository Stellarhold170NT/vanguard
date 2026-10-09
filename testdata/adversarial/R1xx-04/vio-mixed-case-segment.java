// corpus: R1xx-04 vio mixed-case-segment
// lure: capital N in the middle of a kebab segment — /search/all-by-Name next to all-by-condition.
package com.example.magazines;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class MagazineController {

    @GetMapping("/search/all-by-Name")
    public List<MagazineDto> search(@RequestParam String name) {
        return List.of();
    }
}

record MagazineDto(Long id, String name) {
}
