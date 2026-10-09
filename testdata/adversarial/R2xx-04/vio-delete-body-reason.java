// corpus: R2xx-04 vio delete-body-reason
// lure: DELETE with a reason body on the item path — the reason belongs in an audit trail, not a DELETE body.
package com.example.members;

import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class MembershipController {

    @DeleteMapping("/memberships/{id}")
    public void end(@PathVariable Long id, @RequestBody EndReason reason) {
    }
}

record EndReason(String justification, boolean refundDue) {
}
