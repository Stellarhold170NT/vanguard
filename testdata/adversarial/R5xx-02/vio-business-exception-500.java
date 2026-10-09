// corpus: R5xx-02 vio business-exception-500
// lure: an app-owned BookNotFoundException answered with 500 — a business failure as a server fault.
package com.example.books;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

@ControllerAdvice
class BookAdvice {

    @ExceptionHandler(BookNotFoundException.class)
    @ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
    ErrorResponse missing(BookNotFoundException ex) {
        return new ErrorResponse("BOOK_NOT_FOUND", ex.getMessage());
    }
}

class BookNotFoundException extends RuntimeException {
    BookNotFoundException(String message) {
        super(message);
    }
}

record ErrorResponse(String code, String message) {
}
