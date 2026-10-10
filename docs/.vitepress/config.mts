import { defineConfig } from "vitepress";

const ruleSidebar = (group: string, items: [string, string][]) => ({
  text: group,
  collapsed: false,
  items: items.map(([text, link]) => ({ text, link })),
});

const rulesSidebar = [
  {
    text: "Rule Reference",
    collapsed: false,
    items: [{ text: "Catalog Overview", link: "/rules/README" }],
  },
  ruleSidebar("R1xx — Resources & Naming", [
    ["R1xx-01 plural-collection", "/rules/R1xx-01-plural-collection"],
    ["R1xx-02 no-verb-path", "/rules/R1xx-02-no-verb-path"],
    ["R1xx-03 resource-path-pattern", "/rules/R1xx-03-resource-path-pattern"],
    ["R1xx-04 path-casing", "/rules/R1xx-04-path-casing"],
    ["R1xx-05 id-field-naming", "/rules/R1xx-05-id-field-naming"],
  ]),
  ruleSidebar("R2xx — Methods & Verbs", [
    ["R2xx-01 get-no-body", "/rules/R2xx-01-get-no-body"],
    ["R2xx-02 post-creates-201", "/rules/R2xx-02-post-creates-201"],
    ["R2xx-03 patch-partial", "/rules/R2xx-03-patch-partial"],
    ["R2xx-04 delete-no-body", "/rules/R2xx-04-delete-no-body"],
    ["R2xx-05 custom-method-post", "/rules/R2xx-05-custom-method-post"],
    ["R2xx-06 put-full-update", "/rules/R2xx-06-put-full-update"],
  ]),
  ruleSidebar("R3xx — Pagination & Collections", [
    ["R3xx-01 list-paginated", "/rules/R3xx-01-list-paginated"],
    ["R3xx-02 list-envelope", "/rules/R3xx-02-list-envelope"],
    ["R3xx-03 unbounded-page-size", "/rules/R3xx-03-unbounded-page-size"],
  ]),
  ruleSidebar("R4xx — Payload", [
    ["R4xx-01 no-entity-in-payload", "/rules/R4xx-01-no-entity-in-payload"],
    ["R4xx-02 field-casing", "/rules/R4xx-02-field-casing"],
    ["R4xx-03 time-field-standard", "/rules/R4xx-03-time-field-standard"],
  ]),
  ruleSidebar("R5xx — Errors & Status", [
    ["R5xx-01 unified-error-shape", "/rules/R5xx-01-unified-error-shape"],
    ["R5xx-02 no-500-for-business", "/rules/R5xx-02-no-500-for-business"],
    ["R5xx-03 status-semantics", "/rules/R5xx-03-status-semantics"],
  ]),
  ruleSidebar("R6xx — Versioning & gRPC", [
    ["R6xx-01 versioned-path", "/rules/R6xx-01-versioned-path"],
    ["R6xx-02 grpc-standard-methods", "/rules/R6xx-02-grpc-standard-methods"],
  ]),
  ruleSidebar("R6xx-9x — Demo Fixtures", [
    ["R6xx-91 demo-bad-method-name", "/rules/R6xx-91-demo-bad-method-name"],
    ["R6xx-92 demo-no-response-type", "/rules/R6xx-92-demo-no-response-type"],
    ["R6xx-93 demo-get-with-body", "/rules/R6xx-93-demo-get-with-body"],
    ["R6xx-94 demo-legacy-path", "/rules/R6xx-94-demo-legacy-path"],
    ["R6xx-99 demo-trailing-slash", "/rules/R6xx-99-demo-trailing-slash"],
  ]),
];

export default defineConfig({
  title: "Vanguard",
  description: "Source-first API design linter for Java/Spring Boot",
  cleanUrls: true,
  lastUpdated: true,
  base: process.env.DOCS_BASE || "/",
  // Engineering pages reference repository paths outside the site
  // (reports/*, .vanguard.example.yaml, testdata/*); keep the build green.
  ignoreDeadLinks: true,
  srcExclude: ["README.md"],

  head: [
    ["link", { rel: "icon", type: "image/svg+xml", href: "/favicon.svg" }],
    ["meta", { property: "og:title", content: "Vanguard Documentation" }],
    [
      "meta",
      {
        property: "og:description",
        content: "Source-first API design linter for Java/Spring Boot",
      },
    ],
  ],

  themeConfig: {
    logo: "/images/logo.svg",
    siteTitle: "Vanguard",

    nav: [
      { text: "Getting Started", link: "/getting-started/" },
      { text: "Rules", link: "/rules/README" },
      { text: "Architecture", link: "/architecture/" },
      { text: "Guides", link: "/guides/" },
      {
        text: "Reference",
        items: [
          { text: "CLI", link: "/reference/cli" },
          { text: "Configuration", link: "/config" },
          { text: "Installation", link: "/install" },
          { text: "CI Integration", link: "/ci-integration" },
          { text: "Demo", link: "/demo" },
        ],
      },
      { text: "Community", link: "/community/contributing" },
    ],

    sidebar: {
      "/getting-started/": [
        {
          text: "Getting Started",
          items: [
            { text: "Quick Start", link: "/getting-started/" },
            { text: "Installation", link: "/install" },
            { text: "Configuration", link: "/config" },
          ],
        },
        {
          text: "Next Steps",
          items: [
            { text: "Rules", link: "/rules/README" },
            { text: "Architecture", link: "/architecture/" },
            { text: "Guides", link: "/guides/" },
          ],
        },
      ],

      "/rules/": rulesSidebar,

      "/architecture/": [
        {
          text: "Architecture",
          items: [
            { text: "Overview", link: "/architecture/" },
            { text: "Scanning Pipeline & IR", link: "/architecture/pipeline" },
            {
              text: "Rules Engine & Determinism",
              link: "/architecture/rules-engine",
            },
          ],
        },
        {
          text: "Design Docs",
          items: [
            { text: "Charter v1", link: "/charter" },
            { text: "Test Strategy", link: "/test-strategy" },
            { text: "FP/FN Audit Protocol", link: "/fpfn-protocol" },
          ],
        },
      ],

      "/guides/": [
        {
          text: "Guides",
          items: [
            { text: "Overview", link: "/guides/" },
            { text: "CI Integration", link: "/ci-integration" },
            { text: "Tuning & Suppression", link: "/guides/tuning" },
            {
              text: "Case Study: Real-Fire Baseline",
              link: "/guides/case-study",
            },
          ],
        },
      ],

      "/reference/": [
        {
          text: "Reference",
          items: [
            { text: "CLI", link: "/reference/cli" },
            { text: "Configuration", link: "/config" },
            { text: "Installation", link: "/install" },
            { text: "CI Integration", link: "/ci-integration" },
            { text: "Demo", link: "/demo" },
          ],
        },
      ],

      "/config": [
        {
          text: "Reference",
          items: [
            { text: "CLI", link: "/reference/cli" },
            { text: "Configuration", link: "/config" },
            { text: "Installation", link: "/install" },
            { text: "CI Integration", link: "/ci-integration" },
            { text: "Demo", link: "/demo" },
          ],
        },
        {
          text: "Guides",
          items: [
            { text: "Tuning & Suppression", link: "/guides/tuning" },
            { text: "Case Study", link: "/guides/case-study" },
          ],
        },
      ],

      "/community/": [
        {
          text: "Community",
          items: [
            { text: "Contributing", link: "/community/contributing" },
            { text: "Versioning & Releases", link: "/community/versioning" },
          ],
        },
      ],
    },

    editLink: {
      pattern:
        "https://github.com/Stellarhold170NT/vanguard/edit/main/docs/:path",
      text: "Edit this page on GitHub",
    },

    socialLinks: [
      {
        icon: "github",
        link: "https://github.com/Stellarhold170NT/vanguard",
      },
    ],

    footer: {
      message: "Released under the Apache 2.0 License.",
      copyright: "Copyright © 2026 Vanguard Contributors",
    },

    search: {
      provider: "local",
    },

    outline: {
      level: [2, 3],
    },
  },
});
