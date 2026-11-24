import express from "express";

const app = express();
const port = Number(process.env.PORT || 8080);

app.get("/", (_request, response) => {
  response.json({ app: "cloudforge-node-example", message: "deployed by CloudForge" });
});

app.get("/healthz", (_request, response) => response.status(200).send("ok"));

app.listen(port, "0.0.0.0", () => {
  console.log(`node example listening on ${port}`);
});
