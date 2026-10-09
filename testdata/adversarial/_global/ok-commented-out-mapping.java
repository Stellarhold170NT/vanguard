// corpus: _global ok commented-out-mapping
// lure: the whole mapping lives in a // comment — comments are not an API surface, nothing may fire.
package com.example.archive;

import org.springframework.web.bind.annotation.RestController;

@RestController
public class ArchiveController {

    // @GetMapping("/delete/users")
    // public String deleteAll() { return "ok"; }
}
