// corpus: R1xx-03 vio param-first-path
// lure: the path opens with {memberId} — no collection segment anchors the variable.
package com.example.loans;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LoanController {

    @GetMapping("/{memberId}/loans")
    public List<LoanDto> list(@PathVariable Long memberId) {
        return List.of();
    }
}

record LoanDto(Long id, Long memberId) {
}
