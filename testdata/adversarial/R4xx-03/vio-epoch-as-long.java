// corpus: R4xx-03 vio epoch-as-long
// lure: updatedAt as a raw long — an epoch number with no zone or unit contract.
package com.example.devices;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class DeviceController {

    @GetMapping("/devices/{id}")
    public DeviceDto get(@PathVariable Long id) {
        return null;
    }
}

record DeviceDto(Long id, long updatedAt) {
}
