// corpus: R2xx-01 vio get-requestpart
// lure: @RequestPart on a GET — multipart or not, the adapter binds it as a body and the rule speaks.
package com.example.attachments;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestPart;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

@RestController
public class AttachmentController {

    @GetMapping("/attachments")
    public AttachmentDto upload(@RequestPart MultipartFile file) {
        return null;
    }
}

record AttachmentDto(Long id, String name) {
}
