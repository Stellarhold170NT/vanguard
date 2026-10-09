// corpus: R2xx-02 ok post-202-async
// lure: asynchronous creation answering 202 Accepted — the deliberate AIP-133 dispute edge the rule accepts.
package com.example.imports;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ImportController {

    @PostMapping("/import-jobs")
    @ResponseStatus(HttpStatus.ACCEPTED)
    public ImportJobDto enqueue(@RequestBody ImportJobDto job) {
        return job;
    }
}

record ImportJobDto(Long id, String status) {
}
