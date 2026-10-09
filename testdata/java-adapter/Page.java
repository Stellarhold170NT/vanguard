package com.example.books;

import java.util.List;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

@JsonIgnoreProperties(ignoreUnknown = true)
public class Page<T> {

    private List<T> items;

    private int total;

    public List<T> getItems() {
        return items;
    }

    public int getTotal() {
        return total;
    }
}
