// corpus: R5xx-04 vio advice-plus-local-handler
// lure: a global advice plus a controller-local @ExceptionHandler — a second error source beside the envelope.
package com.example.devices;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@ControllerAdvice
class DeviceAdvice {

    @ExceptionHandler(Exception.class)
    @ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
    ErrorResponse fallback(Exception ex) {
        return new ErrorResponse("FALLBACK", ex.getMessage());
    }
}

@RestController
class DeviceController {

    @ExceptionHandler(IllegalArgumentException.class)
    @ResponseStatus(HttpStatus.BAD_REQUEST)
    ErrorResponse invalid(IllegalArgumentException ex) {
        return new ErrorResponse("BAD_REQUEST", ex.getMessage());
    }
}

record ErrorResponse(String code, String message) {
}
