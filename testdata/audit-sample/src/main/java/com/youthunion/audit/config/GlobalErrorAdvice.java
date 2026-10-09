// audit-sample: the error envelope (pattern w1-03 pain 5 — one app-wide
// error contract). A project-owned type so R5xx-01's majority-envelope
// inference is exercisable: 3 of 4 handlers answer this envelope, the
// legacy Map handler deviates.
//
// NOTE: military-youth's real translator answers Spring's ProblemDetail,
// which lives outside the scanned sources — v0.1's resolvable-consensus
// inference then stays silent (documented limitation, w5-04 input). This
// sample uses an in-scope envelope so the rule's predicate is actually
// exercised.
package com.youthunion.audit.config;

import java.util.Map;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

import com.youthunion.audit.service.ResourceNotFoundException;
import com.youthunion.audit.service.YouthUnionException;

@ControllerAdvice
public class GlobalErrorAdvice {

    @ExceptionHandler(ResourceNotFoundException.class)
    @ResponseStatus(HttpStatus.NOT_FOUND)
    public ErrorResponse missing(ResourceNotFoundException ex) {
        return new ErrorResponse("NOT_FOUND", ex.getMessage());
    }

    @ExceptionHandler(IllegalArgumentException.class)
    @ResponseStatus(HttpStatus.BAD_REQUEST)
    public ErrorResponse invalid(IllegalArgumentException ex) {
        return new ErrorResponse("BAD_REQUEST", ex.getMessage());
    }

    @ExceptionHandler(YouthUnionException.class)
    @ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
    public ErrorResponse business(YouthUnionException ex) {
        return new ErrorResponse("BUSINESS_RULE", ex.getMessage());
    }

    @ExceptionHandler(IllegalStateException.class)
    public Map<String, Object> legacy(IllegalStateException ex) {
        return Map.of("error", ex.getMessage());
    }

    public record ErrorResponse(String code, String message) {
    }
}
