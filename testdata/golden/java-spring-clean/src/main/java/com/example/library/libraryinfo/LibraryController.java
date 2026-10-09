package com.example.library.libraryinfo;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/** The clean info endpoint — versioned base, typed response. */
@RestController
@RequestMapping("/api/v1/library")
public class LibraryController {

    @GetMapping
    public LibraryInfo getLibraryInfo() {
        return null;
    }
}
