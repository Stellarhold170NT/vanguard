package com.example.books;

// Un-annotated interface: inventory only, never a candidate. Interface
// methods carry no source modifiers, and method/parameter annotations must
// still be captured.
public interface BookMapper {

    BookDto toDto(Book book);

    @Deprecated
    String legacyName(@Deprecated Long id);
}
