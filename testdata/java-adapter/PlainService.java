package com.example.books;

// A plain service class: no annotations anywhere, so it must never surface
// as an API candidate (w3-01 fixture), while its public methods stay in the
// per-file inventory for w3-02.
public class PlainService {

    public BookDto findById(Long id) {
        return null;
    }

    public void audit(String action) {
    }
}
