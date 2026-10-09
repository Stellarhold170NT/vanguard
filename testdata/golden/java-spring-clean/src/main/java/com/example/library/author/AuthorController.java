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

/** The clean author API (w3-07 golden showcase, clean case). */
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
