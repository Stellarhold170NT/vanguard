// audit-sample: the odd one out (pattern w1-03 §2.1 — RoutesController
// maps bare /api while every other controller uses /api/<service>/…).
// R6xx-03 version-consistency is chartered but NOT registered in v0.1 —
// declared coverage gap in expectations.json.
package com.youthunion.audit.web.rest;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api")
public class RouteIndexResource {

    @GetMapping
    public List<String> index() {
        return List.of("youths", "movement-logs", "training-courses");
    }
}
