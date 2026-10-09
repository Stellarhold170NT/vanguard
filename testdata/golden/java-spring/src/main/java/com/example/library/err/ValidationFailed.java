package com.example.library.err;

/**
 * A validation failure whose name deliberately lacks the Exception suffix —
 * R5xx-02's ownership heuristic stays silent on it even though it is
 * app-declared (documented negative, w3-07 showcase).
 */
public class ValidationFailed extends RuntimeException {

    public ValidationFailed(String message) {
        super(message);
    }
}
