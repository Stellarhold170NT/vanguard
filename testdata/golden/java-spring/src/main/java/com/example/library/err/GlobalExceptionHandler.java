package com.example.library.err;

import java.util.HashMap;
import java.util.Map;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestControllerAdvice;

/**
 * Error advice. Violations on purpose (w3-07 golden showcase):
 *
 * <ul>
 *   <li>handleBusiness — R5xx-02 ERROR (app-declared BusinessException
 *       mapped to 500)</li>
 *   <li>handleValidation — R5xx-01 (a private Map envelope splitting the
 *       error contract; the majority envelope here is ErrorResponse)</li>
 *   <li>handleNotFound — the clean counterpart (same envelope, 4xx)</li>
 * </ul>
 */
@RestControllerAdvice
public class GlobalExceptionHandler {

    @ExceptionHandler(BusinessException.class)
    @ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
    public ErrorResponse handleBusiness(BusinessException ex) {
        return null;
    }

    @ExceptionHandler(BookNotFoundException.class)
    @ResponseStatus(HttpStatus.NOT_FOUND)
    public ErrorResponse handleNotFound(BookNotFoundException ex) {
        return null;
    }

    @ExceptionHandler(ValidationFailed.class)
    public Map<String, Object> handleValidation(ValidationFailed ex) {
        return new HashMap<>();
    }
}
