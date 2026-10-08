package com.example.books;

import java.util.List;

@Deprecated
public class Overloads {

    public BookDto find(Long id) {
        return null;
    }

    public BookDto find(String title) {
        return null;
    }

    public List<BookDto> find(String title, int limit) {
        return null;
    }

    public java.util.List<String> titles() {
        return null;
    }
}
