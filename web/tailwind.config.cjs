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
      },
      boxShadow: {
        panel: "0 18px 40px rgba(15, 23, 42, 0.12)",
      },
    },
  },
  plugins: [],
};
