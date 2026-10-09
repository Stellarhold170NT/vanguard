package com.example.library.book;

import jakarta.validation.constraints.Max;
import org.springframework.data.domain.Page;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

/**
 * The clean book API (w3-07 golden showcase, clean case): paginated capped
 * enveloped list, typed item route, 201 creation, same-type PUT — the
 * whole R1xx/R2xx/R3xx families stay silent.
 */
@RestController
@RequestMapping("/api/v1/books")
public class BookController {

    @GetMapping
    public Page<BookDto> listBooks(@RequestParam int page, @RequestParam @Max(100) int size) {
        return null;
    }

    @GetMapping("/{bookId}")
    public BookDto getBook(@PathVariable Long bookId) {
        return null;
    }

    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public BookDto createBook(@RequestBody BookDto dto) {
        return null;
    }

    @PutMapping("/{bookId}")
    public BookDto updateBook(@RequestBody BookDto dto, @PathVariable Long bookId) {
        return null;
    }
}
