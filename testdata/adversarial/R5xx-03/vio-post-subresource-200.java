// corpus: R5xx-03 vio post-subresource-200
// lure: creating a sub-resource with the implicit 200 — the final segment is a plain resource, so the create shape holds.
package com.example.notes;

import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class NoteController {

    @PostMapping("/members/{memberId}/notes")
    public NoteDto create(@PathVariable Long memberId, @RequestBody NoteDto draft) {
        return draft;
    }
}

record NoteDto(Long id, String body) {
}
