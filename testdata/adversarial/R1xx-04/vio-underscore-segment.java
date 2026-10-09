// corpus: R1xx-04 vio underscore-segment
// lure: snake_case literal /user_profile masquerades as a convention — segments are kebab.
package com.example.profiles;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ProfileController {

    @GetMapping("/user_profile")
    public String get() {
        return "{}";
    }
}
