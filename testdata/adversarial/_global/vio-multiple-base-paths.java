// corpus: _global vio multiple-base-paths
// lure: two base paths on one class — the engine keeps the first and surfaces the rest as a note.
package com.example.multi;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping({"/a/v1", "/b"})
public class MultiBaseController {

    @GetMapping("/items")
    public ItemDto get() {
        return null;
    }
}

record ItemDto(Long id, String label) {
}
