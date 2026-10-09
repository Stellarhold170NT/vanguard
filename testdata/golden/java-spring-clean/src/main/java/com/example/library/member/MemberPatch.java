package com.example.library.member;

/** Dedicated partial payload — a different type from MemberDto (R2xx-03 silence). */
public record MemberPatch(String fullName) {
}
