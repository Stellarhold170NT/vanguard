package com.example.library.libraryinfo;

/** A plain response record for the versioned info endpoint. */
public record LibraryInfo(String name, int bookCount) {
}
