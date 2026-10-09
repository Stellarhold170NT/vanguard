// corpus: R3xx-02 vio set-response
// lure: a Set reads just as bare on the wire as a List — the container type changes nothing.
package com.example.roles;

import java.util.Set;

import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class RoleController {

    @GetMapping("/roles")
    public Set<RoleDto> list(Pageable pageable) {
        return Set.of();
    }
}

record RoleDto(Long id, String code) {
}
