package com.example.library.author;

import jakarta.validation.constraints.Max;
import org.springframework.data.domain.Page;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

/**
 * Author API — the CLEAN counterpart block of the showcase (w3-07): a POST
 * that answers 201 Created (R2xx-02 / R5xx-03 silence) and a paginated,
 * capped, enveloped list (R3xx-01/02/03 silence).
 */
@RestController
@RequestMapping("/api/v1/authors")
public class AuthorController {

    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public AuthorDto createAuthor(@RequestBody AuthorDto dto) {
        return null;
    }

    @GetMapping
    public Page<AuthorDto> listAuthors(@RequestParam int page, @RequestParam @Max(100) int size) {
        return null;
    }
}
