// corpus: R2xx-05 vio delete-cleanup
// lure: DELETE /sessions/cleanup — an action segment on a DELETE; the verb already says removal.
package com.example.sessions;

import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class SessionController {

    @DeleteMapping("/sessions/cleanup")
    public void cleanup() {
    }
}
