package com.example.library.controller;

import com.example.library.member.Member;
import java.util.List;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/admin")
public class AdminController {

    @GetMapping("/members")
    public List<Member> members() {
        return List.of();
    }

    @RequestMapping(method = RequestMethod.DELETE)
    public void purge() {
    }
}
