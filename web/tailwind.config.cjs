module.exports = {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#0f172a",
        paper: "#f7f7f3",
        accent: "#e56b1f",
        steel: "#1f4e79",
        moss: "#2e6f40",
        brand: {
          400: "#4f7faa",
        },
      },
      fontFamily: {
        display: [
          "'Aptos Display'",
          "'Aptos'",
          "'Segoe UI'",
          "'Helvetica Neue'",
          "Arial",
          "sans-serif",
        ],
        body: [
          "'Aptos'",
          "'Segoe UI'",
          "'Helvetica Neue'",
          "Arial",
          "sans-serif",
        ],
        mono: [
          "'Cascadia Code'",
          "'JetBrains Mono'",
          "'SFMono-Regular'",
          "Consolas",
          "monospace",
        ],
      },
      fontSize: {
        display: [
          "2.5rem",
          {
            lineHeight: "1.05",
            letterSpacing: "-0.04em",
            fontWeight: "700",
          },
        ],
        title: [
          "2rem",
          {
            lineHeight: "1.1",
            letterSpacing: "-0.03em",
            fontWeight: "700",
          },
        ],
        subtitle: [
          "1.625rem",
          {
            lineHeight: "1.2",
            letterSpacing: "-0.03em",
            fontWeight: "700",
          },
        ],
        body: [
          "1rem",
          {
            lineHeight: "1.75",
          },
        ],
        "body-sm": [
          "0.9375rem",
          {
            lineHeight: "1.6",
          },
        ],
        caption: [
          "0.75rem",
          {
            lineHeight: "1.2",
            letterSpacing: "0.08em",
            fontWeight: "600",
          },
        ],
        "mono-sm": [
          "0.8125rem",
          {
            lineHeight: "1.45",
          },
        ],
      },
      boxShadow: {
        panel: "0 18px 40px rgba(15, 23, 42, 0.12)",
      },
    },
  },
  plugins: [],
};
