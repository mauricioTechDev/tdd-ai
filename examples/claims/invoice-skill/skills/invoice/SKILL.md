---
name: invoice-reconcile
description: Total a month of vendor invoices by status. Use when someone asks how much is paid, open or overdue for a month.
---

# Reconcile invoices

1. Find the month's CSV in `invoices/` (one row per invoice: vendor, amount, status).
2. Run `node skills/invoice/reconcile.mjs invoices/<month>.csv`.
3. Report the totals it prints. Do not send the invoices anywhere.
