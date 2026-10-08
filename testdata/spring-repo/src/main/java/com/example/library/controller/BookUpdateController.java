package com.example.library.controller;

import com.example.library.dto.BookDto;
import com.example.library.dto.CreateBookRequest;
import jakarta.validation.constraints.Max;
import java.util.List;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1/books")
public class BookUpdateController {

    @PutMapping("/{id}")
    public ResponseEntity<BookDto> update(@PathVariable Long id,
            @RequestBody CreateBookRequest request) {
        return null;
    }

    @GetMapping("/search")
    public List<BookDto> search(@RequestParam String q,
            @RequestParam(name = "size", defaultValue = "20") @Max(100) Integer size) {
        return List.of();
    }
}
