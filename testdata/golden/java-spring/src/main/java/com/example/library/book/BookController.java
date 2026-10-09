package com.example.library.book;

import com.example.library.domain.Book;
import java.util.List;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

/**
 * Book API. Violations on purpose (w3-07 golden showcase — every finding is
 * pinned in the snapshot and mapped in README.md):
 *
 * <ul>
 *   <li>listAllBooks — R3xx-02 (bare List) + R3xx-03 (uncapped size)</li>
 *   <li>getBook — R1xx-04 (path variable {BookId} not lowerCamel)</li>
 *   <li>findBookById — R1xx-02 (CRUD verb in path) + R1xx-03 (flat route)</li>
 *   <li>createBook — R2xx-02 (POST create answering non-201) + R5xx-03
 *       (implicit 200) + R1xx-05/R4xx-02/R4xx-03 via BookDto</li>
 *   <li>importBook — R4xx-01 (ORM entity on the wire)</li>
 * </ul>
 */
@RestController
@RequestMapping("/api/v1/books")
public class BookController {

    @GetMapping
    public List<BookDto> listAllBooks(@RequestParam int page, @RequestParam int size) {
        return null;
    }

    @GetMapping("/{BookId}")
    public BookDto getBook(@PathVariable Long bookId) {
        return null;
    }

    @GetMapping("/find-by-id/{id}")
    public BookDto findBookById(@PathVariable Long id) {
        return null;
    }

    @PostMapping
    public BookDto createBook(@RequestBody BookDto dto) {
        return null;
    }

    @PostMapping("/import")
    public Book importBook(@RequestBody Book entity) {
        return null;
    }
}
