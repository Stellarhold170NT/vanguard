package com.example.payload.web;

import com.example.payload.web.Youth;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1/youth")
public class YouthController {

    @GetMapping("/{id}")
    public Youth get(@PathVariable Long id) {
        return null;
    }
}
