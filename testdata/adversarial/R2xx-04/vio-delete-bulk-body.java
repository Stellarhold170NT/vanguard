// corpus: R2xx-04 vio delete-bulk-body
// lure: DELETE with a typed bulk body — the batch belongs in a path or query ids, not a request body.
package com.example.members;

import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class BulkController {

    @DeleteMapping("/memberships")
    public void endAll(@RequestBody MembershipEndRequest request) {
    }
}

record MembershipEndRequest(java.util.List<Long> membershipIds, String reason) {
}
