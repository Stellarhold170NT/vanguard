// corpus: R4xx-03 vio localdatetime-field
// lure: expiresAt as LocalDateTime — a wall-clock stamp with no zone.
package com.example.licenses;

import java.time.LocalDateTime;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LicenseController {

    @GetMapping("/licenses/{id}")
    public LicenseDto get(@PathVariable Long id) {
        return null;
    }
}

record LicenseDto(Long id, LocalDateTime expiresAt) {
}
