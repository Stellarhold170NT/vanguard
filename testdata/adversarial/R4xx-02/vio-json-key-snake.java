// corpus: R4xx-02 vio json-key-snake
// lure: the Java field is clean camelCase — the @JsonProperty key is what ships, and it is snake_case.
package com.example.members;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import com.fasterxml.jackson.annotation.JsonProperty;

@RestController
public class MemberController {

    @GetMapping("/members/{id}")
    public MemberDto get(@PathVariable Long id) {
        return null;
    }
}

record MemberDto(Long id, @JsonProperty("user_name") String userName) {
}
