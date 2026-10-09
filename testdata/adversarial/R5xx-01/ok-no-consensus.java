// corpus: R5xx-01 ok no-consensus
// lure: two handlers with two different envelope types — no dominant standard, the rule stays silent.
package com.example.workshops;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

@ControllerAdvice
class WorkshopAdvice {

    @ExceptionHandler(NoSuchElementException.class)
    @ResponseStatus(HttpStatus.NOT_FOUND)
    ErrorResponse missing(NoSuchElementException ex) {
        return new ErrorResponse("NOT_FOUND", ex.getMessage());
    }

    @ExceptionHandler(IllegalStateException.class)
    @ResponseStatus(HttpStatus.CONFLICT)
    ProblemDetail conflict(IllegalStateException ex) {
        return new ProblemDetail(409, ex.getMessage());
    }
}

record ErrorResponse(String code, String message) {
}

record ProblemDetail(int status, String detail) {
}
