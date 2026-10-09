// corpus: _global vio nested-class-controller
// lure: the controller is a static nested class — the singular /book/{id} hit proves the walk reaches it.
package com.example.inner;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

class Library {

    @RestController
    static class InnerBookController {

        @GetMapping("/book/{id}")
        public BookDto get(@PathVariable Long id) {
            return null;
        }
    }
}

record BookDto(Long id, String title) {
}
