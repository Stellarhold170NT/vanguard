// corpus: R1xx-03 ok three-level-hierarchy
// lure: three nested levels stay legal — every variable keeps its resource anchor.
package com.example.geo;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class DistrictController {

    @GetMapping("/regions/{regionId}/cities/{cityId}/districts/{districtId}")
    public DistrictDto get(@PathVariable Long regionId, @PathVariable Long cityId,
            @PathVariable Long districtId) {
        return null;
    }
}

record DistrictDto(Long id, String name) {
}
