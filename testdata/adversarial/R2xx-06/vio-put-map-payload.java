// corpus: R2xx-06 vio put-map-payload
// lure: PUT with a Map<String,Object> payload against a typed response — the field bag is the partial tell.
package com.example.profiles;

import java.util.Map;

import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ProfileController {

    @PutMapping("/profiles/{id}")
    public ProfileDto put(@PathVariable Long id, @RequestBody Map<String, Object> fields) {
        return null;
    }
}

record ProfileDto(Long id, String displayName) {
}
