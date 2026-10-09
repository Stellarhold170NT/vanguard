// corpus: R1xx-02 vio delete-verb-path
// lure: @DeleteMapping("/delete") — the verb is stated twice, on the method and in the path.
package com.example.records;

import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class RecordController {

    @DeleteMapping("/delete/{id}")
    public void remove(@PathVariable Long id) {
    }
}
