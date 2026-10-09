// corpus: R4xx-02 ok acronym-camel
// lure: userURL starts lower and stays alphabetic — the lowerCamel read passes it; the rule under-reports on acronyms.
package com.example.links;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LinkController {

    @GetMapping("/links/{id}")
    public LinkDto get(@PathVariable Long id) {
        return null;
    }
}

record LinkDto(Long id, String userURL) {
}
