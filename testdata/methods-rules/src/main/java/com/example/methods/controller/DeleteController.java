package com.example.methods.controller;

import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/api/v1/sprockets")
public class DeleteController {

    @DeleteMapping
    public void deleteMany(@RequestBody List<String> ids) {
    }

    @DeleteMapping("/{id}")
    public void deleteOne(@PathVariable Long id) {
    }
}
