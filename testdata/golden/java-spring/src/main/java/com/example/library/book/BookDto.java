package com.example.library.book;

/**
 * Book DTO. Violations on purpose (w3-07 golden showcase):
 *
 * <ul>
 *   <li>R1xx-05 — mixed id spellings: {@code bookID} (Id suffix) next to
 *       {@code author_id} (_id suffix)</li>
 *   <li>R4xx-02 — snake_case serialized fields {@code author_id},
 *       {@code created_at}</li>
 *   <li>R4xx-03 — time-like field {@code created_at} typed String</li>
 * </ul>
 */
public record BookDto(Long bookID, Long author_id, String created_at, String title) {
}
