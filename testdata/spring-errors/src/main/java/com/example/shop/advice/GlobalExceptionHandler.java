package com.example.shop.advice;

import com.example.shop.dto.ApiError;
import com.example.shop.exception.NotFoundException;
import com.example.shop.service.BusinessException;
import java.util.HashMap;
import java.util.Map;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestControllerAdvice;

@RestControllerAdvice
public class GlobalExceptionHandler {

    // R5xx-02 positive: an app-declared business exception mapped to 500.
    @ExceptionHandler(BusinessException.class)
    @ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
    public ApiError handleBusiness(BusinessException ex) {
        return null;
    }

    // Negative: the same envelope, a 4xx outcome.
    @ExceptionHandler(NotFoundException.class)
    @ResponseStatus(HttpStatus.NOT_FOUND)
    public ApiError handleNotFound(NotFoundException ex) {
        return null;
    }

    // Negative: external exception, no business mapping question.
    @ExceptionHandler(IllegalArgumentException.class)
    public ApiError handleIllegalArgument(IllegalArgumentException ex) {
        return null;
    }

    // R5xx-01 positive: a second, private error structure.
    @ExceptionHandler(jakarta.validation.ValidationException.class)
    public Map<String, Object> handleValidation(jakarta.validation.ValidationException ex) {
        return new HashMap<>();
    }
}
