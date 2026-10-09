# java-adapter fixtures

Parse-test corpus for the Java tree-sitter adapter (`adapters/java`, w3-01).
All files are synthetic (charter §7, Phụ lục B-4: never copy proprietary
code) and exercise one construct each:

| File | Construct under test | Candidate? |
|---|---|---|
| `BookController.java` | annotated class, mapping annotations, annotated params, generic return `ResponseEntity<List<BookDto>>`, `void` return | yes |
| `PlainService.java` | class with **no annotations** — must never be a candidate | no |
| `BookDto.java` | record + annotated record components, class-level annotation with one unnamed argument | yes |
| `BookMapper.java` | interface, modifier-less methods, method and parameter annotations | no |
| `Page.java` | generic class `<T>`, generic field/return `List<T>`, named annotation argument | yes |
| `Nested.java` | nested classes (static, inner, public static) under an un-annotated outer class | nested yes |
| `Overloads.java` | three overloaded methods + qualified generic type `java.util.List<String>` | yes |
| `AnnotatedPojo.java` | field annotations incl. a repeatable-container annotation `@Size.List({...})` with nested annotations, public field with initializer | yes |
| `BookStatus.java` | enum declaration | no |
| `BrokenSyntax.java` | syntax error — must produce a diagnostic, other files keep scanning | excluded |

The expected candidate set (in the order the adapter emits it) is pinned by
`adapters/java/parse_test.go`.
