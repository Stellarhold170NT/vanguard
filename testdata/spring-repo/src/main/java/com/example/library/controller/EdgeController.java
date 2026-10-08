package com.example.library.controller;

import com.example.library.exception.BookNotFoundException;
import org.springframework.web.bind.annotation.*;

@RestController
@WeirdThing
public class EdgeController {

    @CustomGet("/weird")
    public String weird() {
        return "weird";
    }

    @RequestMapping("/anything")
    public void anyVerb() {
    }

    @PostMapping({"/x", "/y"})
    public void multiPath() {
    }

    @ExceptionHandler(BookNotFoundException.class)
    public String local(BookNotFoundException ex) {
        return "local";
    }
}
