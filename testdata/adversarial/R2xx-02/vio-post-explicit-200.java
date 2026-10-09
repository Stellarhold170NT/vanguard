// corpus: R2xx-02 vio post-explicit-200
// lure: the status IS declared — as 200 OK. The verb reads right, the status answers wrong.
package com.example.members;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class MemberCreateController {

    @PostMapping("/members")
    @ResponseStatus(HttpStatus.OK)
    public MemberDto create(@RequestBody MemberDto draft) {
        return draft;
    }
}

record MemberDto(Long id, String name) {
}
