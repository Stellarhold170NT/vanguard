// corpus: R5xx-01 vio second-envelope-class
// lure: ApiError against the dominant ErrorResponse — two envelope classes in one advice.
package com.example.subscriptions;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

@ControllerAdvice
class SubscriptionAdvice {

    @ExceptionHandler(NoSuchElementException.class)
    @ResponseStatus(HttpStatus.NOT_FOUND)
    ErrorResponse missing(NoSuchElementException ex) {
        return new ErrorResponse("NOT_FOUND", ex.getMessage());
    }

    @ExceptionHandler(IllegalStateException.class)
    @ResponseStatus(HttpStatus.CONFLICT)
    ErrorResponse conflict(IllegalStateException ex) {
        return new ErrorResponse("CONFLICT", ex.getMessage());
    }

    @ExceptionHandler(IllegalArgumentException.class)
    @ResponseStatus(HttpStatus.BAD_REQUEST)
    ApiError invalid(IllegalArgumentException ex) {
        return new ApiError(400, ex.getMessage());
    }
}

record ErrorResponse(String code, String message) {
}

record ApiError(int status, String message) {
}
