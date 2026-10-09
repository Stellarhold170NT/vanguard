// corpus: R2xx-01 ok get-header-and-query
// lure: trace header plus query term — @RequestHeader binds from headers, never a body.
package com.example.trace;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class TraceController {

    @GetMapping("/members")
    public List<MemberDto> find(@RequestHeader("X-Trace-Id") String traceId,
            @RequestParam String q) {
        return List.of();
    }
}

record MemberDto(Long id, String name) {
}
