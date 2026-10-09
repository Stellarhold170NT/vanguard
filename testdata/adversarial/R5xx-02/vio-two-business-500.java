// corpus: R5xx-02 vio two-business-500
// lure: two app-owned exceptions both mapped to 500 — the whole advice reports business rules as server faults.
package com.example.orders;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

@ControllerAdvice
class OrderAdvice {

    @ExceptionHandler(OrderNotFoundException.class)
    @ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
    ErrorResponse missing(OrderNotFoundException ex) {
        return new ErrorResponse("ORDER_NOT_FOUND", ex.getMessage());
    }

    @ExceptionHandler(OrderStateException.class)
    @ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
    ErrorResponse state(OrderStateException ex) {
        return new ErrorResponse("ORDER_STATE", ex.getMessage());
    }
}

class OrderNotFoundException extends RuntimeException {
}

class OrderStateException extends RuntimeException {
}

record ErrorResponse(String code, String message) {
}
