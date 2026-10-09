package com.example.library.err;

/** Business exception mapped to 409 Conflict — never 500 (R5xx-02 silence). */
public class BusinessException extends RuntimeException {

    public BusinessException(String message) {
        super(message);
    }
}
