package com.example.library.err;

/** A business exception declared in the scanned repo (R5xx-02 app-owned signal). */
public class BusinessException extends RuntimeException {

    public BusinessException(String message) {
        super(message);
    }
}
