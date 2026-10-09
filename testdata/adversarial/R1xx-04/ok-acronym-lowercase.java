// corpus: R1xx-04 ok acronym-lowercase
// lure: acronym spelled lower-case inside kebab (/api-key) stays consistent; variable stays camel.
package com.example.credentials;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ApiKeyController {

    @GetMapping("/api-keys/{keyId}")
    public ApiKeyDto get(@PathVariable Long keyId) {
        return null;
    }
}

record ApiKeyDto(Long id, String secret) {
}
