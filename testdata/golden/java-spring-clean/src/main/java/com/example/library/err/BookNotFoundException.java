package com.example.library.err;

/** 404 outcome. */
public class BookNotFoundException extends RuntimeException {

    public BookNotFoundException(Long id) {
        super("book " + id + " not found");
    }
}
