package com.example.library.member;

import java.util.List;

/** A named envelope (not a bare array) — keeps the pagination family silent on auto-complete. */
public record MemberSuggestions(List<String> suggestions) {
}
