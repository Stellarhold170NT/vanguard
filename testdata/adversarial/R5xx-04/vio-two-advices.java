// corpus: R5xx-04 vio two-advices
// lure: two @ControllerAdvice classes in one file — two error sources race for the same exception.
package com.example.members;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

@ControllerAdvice
class MemberAdvice {

    @ExceptionHandler(NoSuchElementException.class)
    @ResponseStatus(HttpStatus.NOT_FOUND)
    ErrorResponse missing(NoSuchElementException ex) {
        return new ErrorResponse("NOT_FOUND", ex.getMessage());
    }
}

@ControllerAdvice
class MemberFallbackAdvice {

    @ExceptionHandler(Exception.class)
    @ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
    ErrorResponse fallback(Exception ex) {
        return new ErrorResponse("FALLBACK", ex.getMessage());
    }
}

record ErrorResponse(String code, String message) {
}
