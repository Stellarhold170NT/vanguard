package com.example.library.loan;

import jakarta.validation.constraints.Max;
import java.util.List;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/**
 * Loan API. Violation on purpose (w3-07 golden showcase): the singular
 * collection segment fires R1xx-01 (and the bare List fires R3xx-02 — the
 * pagination parameters and the @Max cap keep R3xx-01/R3xx-03 silent, so
 * this endpoint isolates exactly two rules).
 */
@RestController
@RequestMapping("/api/v1/loan")
public class LoanController {

    @GetMapping
    public List<LoanDto> listLoans(@RequestParam int page, @RequestParam @Max(50) int size) {
        return null;
    }
}
