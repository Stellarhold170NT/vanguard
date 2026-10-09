package com.example.methods.controller;

import com.example.methods.dto.WidgetDto;
import com.example.methods.dto.WidgetRequest;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/api/v1/widgets")
public class CreateController {

    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    public WidgetDto createWidget(@RequestBody WidgetRequest request) {
        return null;
    }

    @PostMapping("/direct")
    public ResponseEntity<WidgetDto> createDirect(@RequestBody WidgetRequest request) {
        return ResponseEntity.ok(null);
    }

    @PostMapping("/validate")
    public void validateWidget(@RequestBody WidgetRequest request) {
    }

    @PostMapping("/matches")
    public List<WidgetDto> findMatches(@RequestBody WidgetRequest request) {
        return null;
    }
}
