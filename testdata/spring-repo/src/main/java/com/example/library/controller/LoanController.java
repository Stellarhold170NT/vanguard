package com.example.library.controller;

import com.example.library.dto.LoanDto;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1/loans")
public class LoanController {

    public record BookStats(long total, long available) {
    }

    @PostMapping("/checkout")
    public ResponseEntity<LoanDto> checkout(@PathVariable Long memberId,
            @PathVariable Long bookId,
            @RequestHeader("X-Request-Id") String requestId) {
        return null;
    }

    @PutMapping("/{id}/return")
    public LoanDto giveBack(@PathVariable Long id) {
        return null;
    }

    @GetMapping("/stats")
    public BookStats loanStats() {
        return null;
    }
}
