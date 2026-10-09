// corpus: R2xx-04 ok delete-etag-header
// lure: conditional delete with an If-Match header — @RequestHeader sits near the body slot but binds elsewhere.
package com.example.documents;

import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class DocumentController {

    @DeleteMapping("/documents/{id}")
    public void delete(@PathVariable Long id, @RequestHeader("If-Match") String etag) {
    }
}
