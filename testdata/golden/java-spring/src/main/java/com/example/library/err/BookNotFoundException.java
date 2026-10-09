package com.example.library.err;

/** 404 outcome — the clean counterpart of R5xx-02. */
public class BookNotFoundException extends RuntimeException {

    public BookNotFoundException(Long id) {
        super("book " + id + " not found");
    }
}
