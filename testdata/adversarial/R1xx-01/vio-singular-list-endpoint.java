// corpus: R1xx-01 vio singular-list-endpoint
// lure: list route whose final segment is singular while the response is a page of items.
package com.example.training;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class TrainingPlanController {

    @GetMapping("/training-plan")
    public List<TrainingPlanDto> list() {
        return List.of();
    }
}

record TrainingPlanDto(Long id, String name) {
}
