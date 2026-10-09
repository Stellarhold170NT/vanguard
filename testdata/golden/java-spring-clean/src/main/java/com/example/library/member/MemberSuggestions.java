package com.example.library.member;

import java.util.List;

/** A named envelope for the suggestion endpoint. */
public record MemberSuggestions(List<String> suggestions) {
}
