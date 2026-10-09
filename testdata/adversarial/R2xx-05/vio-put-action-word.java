// corpus: R2xx-05 vio put-action-word
// lure: PUT /statistics/calculate — the action verb rides a full-replacement verb.
package com.example.statistics;

import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class StatsController {

    @PutMapping("/statistics/calculate")
    public StatsDto calculate() {
        return null;
    }
}

record StatsDto(long members, long loans) {
}
