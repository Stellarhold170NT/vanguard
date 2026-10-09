// corpus: R5xx-04 vio two-envelope-classes
// lure: one advice answering two envelope types across handlers — the app-level envelope split.
package com.example.reports;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

@ControllerAdvice
class ReportAdvice {

    @ExceptionHandler(NoSuchElementException.class)
    @ResponseStatus(HttpStatus.NOT_FOUND)
    ErrorResponse missing(NoSuchElementException ex) {
        return new ErrorResponse("NOT_FOUND", ex.getMessage());
    }

    @ExceptionHandler(IllegalStateException.class)
    @ResponseStatus(HttpStatus.CONFLICT)
    ApiError conflict(IllegalStateException ex) {
        return new ApiError(409, ex.getMessage());
    }
}

record ErrorResponse(String code, String message) {
}

record ApiError(int status, String message) {
}
