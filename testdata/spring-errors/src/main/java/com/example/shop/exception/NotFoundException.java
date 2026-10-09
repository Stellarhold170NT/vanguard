package com.example.shop.exception;

/** The referenced resource does not exist — mapped to 404 below. */
public class NotFoundException extends RuntimeException {

    public NotFoundException(String message) {
        super(message);
    }
}
