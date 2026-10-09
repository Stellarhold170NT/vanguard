// corpus: R3xx-04 vio params-vs-pageable
// lure: page/size parameters on one list, a Pageable argument on the next — two input dialects.
package com.example.training;

import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class TrainingController {

    @GetMapping("/members")
    public Page<MemberDto> members(@RequestParam int page, @RequestParam int size) {
        return Page.empty();
    }

    @GetMapping("/trainees")
    public Page<TraineeDto> trainees(Pageable pageable) {
        return Page.empty();
    }
}

record MemberDto(Long id, String name) {
}

record TraineeDto(Long id, String cohort) {
}
