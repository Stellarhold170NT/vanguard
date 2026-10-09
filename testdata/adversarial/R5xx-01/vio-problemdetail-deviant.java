// corpus: R5xx-01 vio problemdetail-deviant
// lure: the app standard is ErrorResponse; one handler drifts to ProblemDetail — a second envelope on the same advice.
package com.example.licenses;

import org.springframework.http.HttpStatus;
import org.springframework.http.ProblemDetail;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

@ControllerAdvice
class LicenseAdvice {

    @ExceptionHandler(NoSuchElementException.class)
    @ResponseStatus(HttpStatus.NOT_FOUND)
    ErrorResponse missing(NoSuchElementException ex) {
        return new ErrorResponse("LICENSE_NOT_FOUND", ex.getMessage());
    }

    @ExceptionHandler(IllegalStateException.class)
    @ResponseStatus(HttpStatus.CONFLICT)
    ErrorResponse conflict(IllegalStateException ex) {
        return new ErrorResponse("LICENSE_CONFLICT", ex.getMessage());
    }

    @ExceptionHandler(IllegalArgumentException.class)
    ProblemDetail invalid(IllegalArgumentException ex) {
        return ProblemDetail.forStatusAndDetail(HttpStatus.BAD_REQUEST, ex.getMessage());
    }
}

record ErrorResponse(String code, String message) {
}
