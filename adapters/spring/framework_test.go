package spring

import (
	"reflect"
	"testing"

	"github.com/Stellarhold170NT/vanguard/adapters/java"
)

func TestFrameworkFromPom(t *testing.T) {
	pom := []byte(`<?xml version="1.0"?>
<project>
  <parent>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-parent</artifactId>
    <version>3.2.5</version>
  </parent>
  <dependencies>
    <dependency>
      <groupId>org.springdoc</groupId>
      <artifactId>springdoc-openapi-starter-webmvc-ui</artifactId>
      <version>2.5.0</version>
    </dependency>
  </dependencies>
</project>`)
	got := frameworkFromPom(pom)
	want := frameworkInfo{framework: "spring-boot", version: "3.2.5", springdoc: true}
	if got != want {
		t.Errorf("frameworkFromPom = %+v, want %+v", got, want)
	}
}

func TestFrameworkFromPomPropertyVersion(t *testing.T) {
	pom := []byte(`<project>
  <properties>
    <spring-boot.version>4.0.2</spring-boot.version>
  </properties>
</project>`)
	got := frameworkFromPom(pom)
	if got != (frameworkInfo{framework: "spring-boot", version: "4.0.2"}) {
		t.Errorf("frameworkFromPom = %+v", got)
	}
}

func TestFrameworkFromPomDependencyVersion(t *testing.T) {
	pom := []byte(`<project>
  <dependencies>
    <dependency>
      <groupId>org.springframework.boot</groupId>
      <artifactId>spring-boot-starter-web</artifactId>
      <version>3.1.0</version>
    </dependency>
  </dependencies>
</project>`)
	got := frameworkFromPom(pom)
	if got != (frameworkInfo{framework: "spring-boot", version: "3.1.0"}) {
		t.Errorf("frameworkFromPom = %+v", got)
	}
}

func TestFrameworkFromPomUnrelated(t *testing.T) {
	pom := []byte(`<project><dependencies><dependency>
  <groupId>com.google.guava</groupId><artifactId>guava</artifactId>
</dependency></dependencies></project>`)
	if got := frameworkFromPom(pom); got != (frameworkInfo{}) {
		t.Errorf("frameworkFromPom = %+v, want zero", got)
	}
}

func TestFrameworkFromGradle(t *testing.T) {
	gradle := []byte(`plugins {
    id 'org.springframework.boot' version '3.1.2'
}
dependencies {
    implementation 'org.springdoc:springdoc-openapi-starter-webmvc-ui:2.2.0'
}`)
	got := frameworkFromGradle(gradle)
	want := frameworkInfo{framework: "spring-boot", version: "3.1.2", springdoc: true}
	if got != want {
		t.Errorf("frameworkFromGradle = %+v, want %+v", got, want)
	}
}

func TestFrameworkFromGradleGroovyVariable(t *testing.T) {
	gradle := []byte(`ext {
    springBootVersion = '2.7.18'
}`)
	got := frameworkFromGradle(gradle)
	if got != (frameworkInfo{framework: "spring-boot", version: "2.7.18"}) {
		t.Errorf("frameworkFromGradle = %+v", got)
	}
}

func TestDetectFrameworkAnnotationsOnly(t *testing.T) {
	// A repo with Spring annotations but no build file still pins the
	// framework — the version just stays empty.
	res := &java.Result{Files: []*java.File{{
		Types: []*java.Class{{
			Name:        "A",
			Annotations: []java.Annotation{{Name: "RestController", Raw: "@RestController"}},
		}},
	}}}
	info := detectFramework(".", []string{"src/A.java"}, res)
	if info != (frameworkInfo{framework: "spring-boot"}) {
		t.Errorf("detectFramework = %+v, want spring-boot (annotation signal)", info)
	}
}

func TestDetectFrameworkNothing(t *testing.T) {
	if got := detectFramework(".", []string{"README.md"}, nil); got != (frameworkInfo{}) {
		t.Errorf("detectFramework = %+v, want zero", got)
	}
}

func TestFindSpecFiles(t *testing.T) {
	files := []string{
		"pom.xml",
		"src/main/resources/openapi.yaml",
		"api/openapi-v2.json",
		"docs/swagger.json",
		"README.md",
		"src/OpenAPI.java",
	}
	got := findSpecFiles(files)
	want := []string{"src/main/resources/openapi.yaml", "api/openapi-v2.json", "docs/swagger.json"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findSpecFiles = %v, want %v (input order)", got, want)
	}
}
