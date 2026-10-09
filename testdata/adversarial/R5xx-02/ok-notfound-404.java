// corpus: R5xx-02 ok notfound-404
// lure: the same app-owned exception mapped to 404 — the right class of status for a missing resource.
package com.example.books;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

@ControllerAdvice
class BookAdvice {

    @ExceptionHandler(BookNotFoundException.class)
    @ResponseStatus(HttpStatus.NOT_FOUND)
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
