package com.example.methods.controller;

import com.example.methods.dto.BookDto;
import com.example.methods.dto.CodeDto;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/v1/books")
public class ActionController {

    @GetMapping("/{id}/generate-code")
    public CodeDto generateCode(@PathVariable Long id) {
        return null;
    }

    @PostMapping("/{id}:archive")
    public BookDto archive(@PathVariable Long id) {
        return null;
    }

    @GetMapping("/{id}")
    public BookDto get(@PathVariable Long id) {
        return null;
    }
}
