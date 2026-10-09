package com.example.library.err;

import java.util.List;

/** The app's one error envelope — every handler answers with it. */
public class ErrorResponse {

    private int code;

    private String message;

    private List<String> details;

    public int getCode() {
        return code;
    }

    public String getMessage() {
        return message;
    }

    public List<String> getDetails() {
        return details;
    }
}
