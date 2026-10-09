package com.example.payload.customer;

import com.example.payload.customer.Customer;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1/customers")
public class CustomerController {

    @GetMapping("/current")
    public Customer current() {
        return null;
    }
}
