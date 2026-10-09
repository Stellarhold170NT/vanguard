// audit-sample: app-owned not-found exception — handled as 404 (the OK
// contrast for R5xx-02: a business exception with a non-500 status).
package com.youthunion.audit.service;

public class ResourceNotFoundException extends RuntimeException {

    public ResourceNotFoundException(String message) {
        super(message);
    }
}
