package com.example.books;

import java.util.List;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1/books")
public class BookController {

    @GetMapping
    public ResponseEntity<List<BookDto>> list() {
        return ResponseEntity.ok().build();
    }

    @PostMapping
    public BookDto create(@RequestBody BookDto book) {
        return book;
    }

    @GetMapping("/{id}")
    public BookDto get(@PathVariable Long id) {
        return null;
    }

    @DeleteMapping("/{id}")
    public void delete(@PathVariable Long id) {
    }
}
