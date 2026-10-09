// audit-sample: the auth controller (pattern w1-03 — VauthzController
// under the service prefix): a login action the create-heuristic reads as
// a create, an unpaginated permission list, a sync action that stays
// silent everywhere.
package com.youthunion.audit.web.rest;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import com.youthunion.audit.dto.AuthTokenDto;
import com.youthunion.audit.dto.PermissionDto;

@RestController
@RequestMapping("/api/youth-union/auth")
public class AuthResource {

    @PostMapping("/login")
    public AuthTokenDto login() {
        return new AuthTokenDto("token", 0);
    }

    @GetMapping("/permissions")
    public List<PermissionDto> permissions() {
        return List.of();
    }

    @PostMapping("/sync-departments")
    public void syncDepartments() {
    }
}
