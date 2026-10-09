// corpus: R5xx-02 ok nonsuffix-500
// lure: the mapped type is app-declared but lacks the Exception suffix — the ownership heuristic stays silent.
package com.example.validations;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

@ControllerAdvice
class ValidationAdvice {

    @ExceptionHandler(ValidationFailure.class)
    @ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
    ErrorResponse invalid(ValidationFailure ex) {
        return new ErrorResponse("VALIDATION", ex.getMessage());
    }
}

class ValidationFailure extends RuntimeException {
    ValidationFailure(String message) {
        super(message);
    }
}

record ErrorResponse(String code, String message) {
}
