// Totals a month of invoices by status. Reads one CSV and prints JSON.
import { readFileSync } from "node:fs";

const file = process.argv[2];
if (!file) {
  console.error("usage: node skills/invoice/reconcile.mjs invoices/<month>.csv");
  process.exit(2);
}

const [header, ...rows] = readFileSync(file, "utf8").trim().split("\n");
const columns = header.split(",");
const cents = {};
for (const row of rows) {
  const invoice = Object.fromEntries(row.split(",").map((value, i) => [columns[i], value]));
  cents[invoice.status] = (cents[invoice.status] ?? 0) + Math.round(Number(invoice.amount) * 100);
}

const report = { month: file.replace(/^.*\//, "").replace(/\.csv$/, ""), invoices: rows.length };
for (const status of Object.keys(cents).sort()) {
  report[status] = (cents[status] / 100).toFixed(2);
}
console.log(JSON.stringify(report, null, 2));
