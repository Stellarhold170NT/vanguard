package com.example.library.controller;

import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;

@Controller
@RequestMapping("/legacy")
public class LegacyViewController {

    @GetMapping
    public String home() {
        return "home";
    }

    @GetMapping("/books/{id}")
    public String book(@PathVariable Long id) {
        return "book";
    }
}
