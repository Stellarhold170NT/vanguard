// corpus: R5xx-04 vio advice-plus-rest-wrapper
// lure: an error advice plus a second @RestControllerAdvice wrapping successes — two envelopes to parse.
package com.example.loans;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestControllerAdvice;

@ControllerAdvice
class LoanErrorAdvice {

    @ExceptionHandler(IllegalStateException.class)
    @ResponseStatus(HttpStatus.CONFLICT)
    ErrorResponse conflict(IllegalStateException ex) {
        return new ErrorResponse("CONFLICT", ex.getMessage());
    }
}

@RestControllerAdvice
class LoanResponseWrapper {

    @ExceptionHandler(IllegalArgumentException.class)
    @ResponseStatus(HttpStatus.BAD_REQUEST)
    WrappedResponse wrap(IllegalArgumentException ex) {
        return new WrappedResponse("error", ex.getMessage());
    }
}

record ErrorResponse(String code, String message) {
}

record WrappedResponse(String kind, String message) {
}
