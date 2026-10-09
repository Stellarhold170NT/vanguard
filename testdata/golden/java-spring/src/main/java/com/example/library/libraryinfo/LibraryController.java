package com.example.library.libraryinfo;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/**
 * Library info — the CLEAN counterpart of R6xx-01 (a versioned service
 * base) and of R6xx-92 (a typed response), w3-07 golden showcase.
 */
@RestController
@RequestMapping("/api/v1/library")
public class LibraryController {

    @GetMapping
    public LibraryInfo getLibraryInfo() {
        return null;
    }
}
