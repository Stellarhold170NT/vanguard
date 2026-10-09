package com.example.library.member;

/** Dedicated update payload — deliberately a different type from MemberDto (R2xx-06 showcase). */
public record MemberUpdateRequest(String fullName, String email) {
}
