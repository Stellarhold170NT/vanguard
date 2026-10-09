// corpus: _global vio uppercase-path-variable
// lure: /trainings/{YouthId} — the casing read must treat the variable as a variable, not a literal.
package com.example.youth;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class YouthTrainingController {

    @GetMapping("/trainings/{YouthId}")
    public TrainingDto get(@PathVariable Long youthId) {
        return null;
    }
}

record TrainingDto(Long id, String title) {
}
