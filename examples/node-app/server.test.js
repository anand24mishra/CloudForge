import test from "node:test";
import assert from "node:assert/strict";

test("example metadata is stable", () => {
  assert.equal("cloudforge-node-example", "cloudforge-node-example");
});
