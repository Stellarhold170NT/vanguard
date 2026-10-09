// corpus: R5xx-02 vio advice-class-500
// lure: @ResponseStatus(INTERNAL_SERVER_ERROR) on the advice class — every handler inherits the server-fault status.
package com.example.trainings;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ControllerAdvice;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.ResponseStatus;

@ControllerAdvice
@ResponseStatus(HttpStatus.INTERNAL_SERVER_ERROR)
class TrainingAdvice {

    @ExceptionHandler(TrainingFullException.class)
    ErrorResponse full(TrainingFullException ex) {
        return new ErrorResponse("TRAINING_FULL", ex.getMessage());
    }
}

class TrainingFullException extends RuntimeException {
    TrainingFullException(String message) {
        super(message);
    }
}

record ErrorResponse(String code, String message) {
}
