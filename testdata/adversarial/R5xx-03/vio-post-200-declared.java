// corpus: R5xx-03 vio post-200-declared
// lure: the status is explicit and still wrong — 200 OK on a creating POST.
package com.example.badges;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BadgeController {

    @PostMapping("/badges")
    @ResponseStatus(HttpStatus.OK)
    public BadgeDto create(@RequestBody BadgeDto draft) {
        return draft;
    }
}

record BadgeDto(Long id, String label) {
}
