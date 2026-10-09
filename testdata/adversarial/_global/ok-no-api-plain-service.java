// corpus: _global ok no-api-plain-service
// lure: a plain @Service class with no mappings — no API surface, zero findings, exit 0.
package com.example.membership;

import org.springframework.stereotype.Service;

@Service
public class MembershipService {

    public boolean isActive(Long memberId) {
        return false;
    }

    public String label(Long memberId) {
        return "inactive";
    }
}
