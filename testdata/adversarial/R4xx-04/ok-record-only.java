// corpus: R4xx-04 ok record-only
// lure: every payload type is a plain record — one convention across the controller.
package com.example.geo;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class RegionController {

    @GetMapping("/regions/{id}")
    public RegionRecord get(@PathVariable Long id) {
        return null;
    }

    @GetMapping("/regions/{id}/summary")
    public RegionSummaryRecord summary(@PathVariable Long id) {
        return null;
    }
}

record RegionRecord(Long id, String name) {
}

record RegionSummaryRecord(Long id, String name, int cities) {
}
