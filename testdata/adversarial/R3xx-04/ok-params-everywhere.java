// corpus: R3xx-04 ok params-everywhere
// lure: both endpoints take page/size parameters and answer Page<T> — consistent input and output.
package com.example.members;

import org.springframework.data.domain.Page;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class MemberController {

    @GetMapping("/members")
    public Page<MemberDto> members(@RequestParam int page, @RequestParam int size) {
        return Page.empty();
    }

    @GetMapping("/trainers")
    public Page<TrainerDto> trainers(@RequestParam int page, @RequestParam int size) {
        return Page.empty();
    }
}

record MemberDto(Long id, String name) {
}

record TrainerDto(Long id, String specialty) {
}
