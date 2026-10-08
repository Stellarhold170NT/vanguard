package com.example.books;

import com.fasterxml.jackson.annotation.JsonProperty;
import jakarta.validation.constraints.Max;
import jakarta.validation.constraints.Size;

@Deprecated
public class AnnotatedPojo {

    @JsonProperty("full_name")
    private String fullName;

    @Max(100)
    public int limit = 10;

    @Size.List({@Size(max = 10), @Size(min = 1)})
    private String code;

    public String getFullName() {
        return fullName;
    }
}
