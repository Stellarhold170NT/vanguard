// corpus: R2xx-04 vio delete-body-list
// lure: DELETE with @RequestBody List<String> — bulk deletes ride query parameters, not bodies.
package com.example.records;

import java.util.List;

import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class RecordPurgeController {

    @DeleteMapping("/records")
    public void purge(@RequestBody List<String> publicIds) {
    }
}
