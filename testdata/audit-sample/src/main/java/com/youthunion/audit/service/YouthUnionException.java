// audit-sample: app-owned business exception (R5xx-02's "app-owned +
// Exception suffix" signal — package resolves from the scan).
package com.youthunion.audit.service;

public class YouthUnionException extends RuntimeException {

    public YouthUnionException(String message) {
        super(message);
    }
}
