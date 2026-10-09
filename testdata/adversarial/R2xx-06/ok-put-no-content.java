// corpus: R2xx-06 ok put-no-content
// lure: full-replacement PUT answering 204 No Content — no response type, the heuristic stays silent.
package com.example.settings;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class SettingController {

    @PutMapping("/settings/{id}")
    @ResponseStatus(HttpStatus.NO_CONTENT)
    public void put(@PathVariable Long id, @RequestBody SettingDto full) {
    }
}

record SettingDto(Long id, String theme) {
}
