package com.example.payload.order;

import java.time.Instant;

public record OrderResponse(
        Long id,
        String code,
        Instant placedAt) {
}
