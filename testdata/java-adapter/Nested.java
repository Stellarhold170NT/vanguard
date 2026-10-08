package com.example.books;

// Outer class without annotations: not a candidate itself, but its
// annotated nested classes are (one static nested, one inner, one public
// static nested).
public class Nested {

    private String outerField;

    @Deprecated
    static class StaticNested {
        public String name() {
            return "static";
        }
    }

    class Inner {
        public String greet() {
            return "hi";
        }
    }

    @Deprecated
    public static class PublicNested {
        public int value() {
            return 1;
        }
    }
}
