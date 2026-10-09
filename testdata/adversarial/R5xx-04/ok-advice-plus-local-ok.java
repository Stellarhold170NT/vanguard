// corpus: R5xx-04 ok advice-plus-local-ok
// lure: a controller-local handler next to a global advice — the scoped pattern the rule accepts.
package com.example.inventory;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@ControllerAdvice
class InventoryAdvice {

    @ExceptionHandler(Exception.class)
    @ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
    ErrorResponse fallback(Exception ex) {
        return new ErrorResponse("FALLBACK", ex.getMessage());
    }
}

@RestController
class ItemController {

    @ExceptionHandler(NumberFormatException.class)
    @ResponseStatus(HttpStatus.BAD_REQUEST)
    ErrorResponse badNumber(NumberFormatException ex) {
        return new ErrorResponse("BAD_NUMBER", ex.getMessage());
    }
}

record ErrorResponse(String code, String message) {
}
