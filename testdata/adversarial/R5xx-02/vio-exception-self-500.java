// corpus: R5xx-02 vio exception-self-500
// lure: the status hides on the exception class itself — the handler stays clean while the type pins 500.
package com.example.licenses;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

@ControllerAdvice
class LicenseAdvice {

    @ExceptionHandler(LicenseExpiredException.class)
    ErrorResponse expired(LicenseExpiredException ex) {
        return new ErrorResponse("LICENSE_EXPIRED", ex.getMessage());
    }
}

@ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
class LicenseExpiredException extends RuntimeException {
    LicenseExpiredException(String message) {
        super(message);
    }
}

record ErrorResponse(String code, String message) {
}
