// corpus: R1xx-04 vio pascal-param
// lure: {YouthId} starts upper-case — path variables are lowerCamelCase.
package com.example.youth;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class YouthController {

    @GetMapping("/trainings/{YouthId}")
    public TrainingDto get(@PathVariable Long youthId) {
        return null;
    }
}

record TrainingDto(Long id, String title) {
}
