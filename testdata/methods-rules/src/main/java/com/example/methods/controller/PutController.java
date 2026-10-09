package com.example.methods.controller;

import com.example.methods.dto.CogDto;
import com.example.methods.dto.CogUpdateRequest;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/v1/cogs")
public class PutController {

    @PutMapping("/{id}")
    public ResponseEntity<CogDto> partialPut(@PathVariable Long id, @RequestBody CogUpdateRequest request) {
        return ResponseEntity.ok(null);
    }

    @PutMapping("/{id}/full")
    public CogDto fullPut(@PathVariable Long id, @RequestBody CogDto body) {
        return null;
    }
}
