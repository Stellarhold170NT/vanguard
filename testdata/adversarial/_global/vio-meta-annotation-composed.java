// corpus: _global vio meta-annotation-composed
// lure: @MyGet composed from @GetMapping + @RequestBody — v0.1 reads direct annotations only, the composed form is skipped.
package com.example.meta;

import java.lang.annotation.ElementType;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@Target(ElementType.METHOD)
@Retention(java.lang.annotation.RetentionPolicy.RUNTIME)
@GetMapping
public @interface MyGet {
    String[] value() default {};
}

@RestController
public class MetaController {

    @MyGet("/meta-books")
    public MetaDto search(@RequestBody MetaQuery query) {
        return null;
    }
}

record MetaQuery(String term) {
}

record MetaDto(Long id, String title) {
}
