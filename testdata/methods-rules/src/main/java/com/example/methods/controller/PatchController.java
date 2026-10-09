package com.example.methods.controller;

import com.example.methods.dto.GadgetDto;
import com.example.methods.dto.GadgetStatusUpdate;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/v1/gadgets")
public class PatchController {

    @PatchMapping("/{id}")
    public GadgetDto fullPatch(@PathVariable Long id, @RequestBody GadgetDto body) {
        return null;
    }

    @PatchMapping("/{id}/status")
    public GadgetDto patchStatus(@PathVariable Long id, @RequestBody GadgetStatusUpdate update) {
        return null;
    }
}
