package com.example.methods.controller;

import com.example.methods.dto.QueryDto;
import com.example.methods.dto.QueryRequest;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/api/v1/queries")
public class GetBodyController {

    @GetMapping("/auto-complete")
    public QueryDto autoComplete(@RequestBody QueryRequest request) {
        return null;
    }

    @GetMapping("/{id}")
    public QueryDto get(@PathVariable Long id) {
        return null;
    }

    @GetMapping("/by-title")
    public List<QueryDto> byTitle(@RequestParam String title) {
        return null;
    }
}
