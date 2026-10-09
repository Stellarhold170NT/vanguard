package com.example.shop.controller;

import com.example.shop.dto.OrderDto;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/v1/orders")
public class OrderController {

    // R5xx-03 positive: an explicit 200 on a create-shaped POST.
    @PostMapping
    @ResponseStatus(HttpStatus.OK)
    public OrderDto create(@RequestBody CreateOrderRequest request) {
        return null;
    }

    // R5xx-03 positive: the implicit 200 (no status declared at all).
    @PostMapping("/v2")
    public OrderDto createV2(@RequestBody CreateOrderRequest request) {
        return null;
    }

    // Negative: the correct 201.
    @PostMapping("/draft")
    @ResponseStatus(HttpStatus.CREATED)
    public OrderDto createDraft(@RequestBody CreateOrderRequest request) {
        return null;
    }

    // Negative: an action endpoint, not a creation.
    @PostMapping("/{id}/cancel")
    @ResponseStatus(HttpStatus.OK)
    public OrderDto cancel(@PathVariable Long id) {
        return null;
    }

    // Negative: a plain read.
    @GetMapping("/{id}")
    public OrderDto get(@PathVariable Long id) {
        return null;
    }
}
