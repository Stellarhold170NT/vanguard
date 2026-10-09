package com.example.library.loan;

import org.springframework.data.domain.Page;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

/** The clean loan API — plural collection, Pageable, enveloped list. */
@RestController
@RequestMapping("/api/v1/loans")
public class LoanController {

    @GetMapping
    public Page<LoanDto> listLoans(org.springframework.data.domain.Pageable pageable) {
        return null;
    }

    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public LoanDto createLoan(@RequestBody LoanDto dto) {
        return null;
    }
}
