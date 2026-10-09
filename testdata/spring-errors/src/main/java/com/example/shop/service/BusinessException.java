package com.example.shop.service;

/** A business rule the caller violated — a 4xx outcome, never a 500. */
public class BusinessException extends RuntimeException {

    public BusinessException(String message) {
        super(message);
    }
}
