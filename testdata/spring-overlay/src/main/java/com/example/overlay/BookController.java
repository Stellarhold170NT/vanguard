package com.example.overlay;

import java.util.List;

import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/books")
public class BookController {

    @GetMapping
    public List<BookDto> list() {
        return List.of();
    }

    @GetMapping("/{id}")
    public BookDto get(@PathVariable Long id) {
        return null;
    }

    @PostMapping
    public BookDto create(@RequestBody BookDto book) {
        return null;
    }
}
