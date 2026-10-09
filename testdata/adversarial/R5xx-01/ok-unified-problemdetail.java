// corpus: R5xx-01 ok unified-problemdetail
// lure: every handler answers the same ProblemDetail record — one envelope, nothing to report.
package com.example.reservations;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

@ControllerAdvice
class ReservationAdvice {

    @ExceptionHandler(NoSuchElementException.class)
    @ResponseStatus(HttpStatus.NOT_FOUND)
    ProblemDetail missing(NoSuchElementException ex) {
        return new ProblemDetail(404, ex.getMessage());
    }

    @ExceptionHandler(IllegalStateException.class)
    @ResponseStatus(HttpStatus.CONFLICT)
    ProblemDetail conflict(IllegalStateException ex) {
        return new ProblemDetail(409, ex.getMessage());
    }

    @ExceptionHandler(IllegalArgumentException.class)
    @ResponseStatus(HttpStatus.BAD_REQUEST)
    ProblemDetail invalid(IllegalArgumentException ex) {
        return new ProblemDetail(400, ex.getMessage());
    }
}

record ProblemDetail(int status, String detail) {
}
