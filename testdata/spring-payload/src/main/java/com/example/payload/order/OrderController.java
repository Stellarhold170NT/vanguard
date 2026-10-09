package com.example.payload.order;

import java.util.List;

import com.example.payload.order.OrderResponse;
import org.springframework.data.domain.Page;
import org.springframework.data.domain.Pageable;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1/orders")
public class OrderController {

    @GetMapping
    public List<OrderResponse> listOrders() {
        return null;
    }

    @GetMapping("/paged")
    public Page<OrderResponse> page(Pageable pageable) {
        return null;
    }

    @GetMapping("/search")
    public List<OrderResponse> search(@RequestParam("size") int size) {
        return null;
    }
}
