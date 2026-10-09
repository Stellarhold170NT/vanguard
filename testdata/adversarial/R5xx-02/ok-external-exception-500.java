// corpus: R5xx-02 ok external-exception-500
// lure: 500 over an undeclared JDK exception — the app does not own it, so the heuristic stays silent.
package com.example.parsing;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

@ControllerAdvice
class ParseAdvice {

    @ExceptionHandler(IllegalStateException.class)
    @ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
    ErrorResponse unexpected(IllegalStateException ex) {
        return new ErrorResponse("UNEXPECTED", ex.getMessage());
    }
}

record ErrorResponse(String code, String message) {
}
