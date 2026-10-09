package com.example.payload.audit;

import java.util.List;

import com.example.payload.dto.AuditDto;
import jakarta.validation.constraints.Max;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1/audits")
public class AuditController {

    @GetMapping("/recent")
    public List<AuditDto> recent(@RequestParam("limit") @Max(200) int limit) {
        return null;
    }

    @GetMapping("/{id}")
    public AuditDto audit(@PathVariable Long id) {
        return null;
    }
}
