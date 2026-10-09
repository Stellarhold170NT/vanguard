package com.example.shop.dto;

import java.util.List;

/** The app's one error envelope — every handler must answer with it. */
public class ApiError {

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
