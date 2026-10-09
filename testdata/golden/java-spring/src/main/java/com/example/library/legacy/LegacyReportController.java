package com.example.library.legacy;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/**
 * Legacy reports — R6xx-94 (the /legacy/ segment) and, with R6xx-01 opted
 * in via .vanguard.yaml, the unversioned base path (R6xx-01) — w3-07
 * golden showcase.
 */
@RestController
@RequestMapping("/legacy/reports")
public class LegacyReportController {

    @GetMapping("/{reportId}")
    public ReportDto getReport(@PathVariable Long reportId) {
        return null;
    }
}
