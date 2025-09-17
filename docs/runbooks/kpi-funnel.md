# KPI Funnel Runbook (Quotes → Labels → Shipments)

Symptoms:
- Conversion ratios drop (Labels/Quotes < 20%, Shipments/Labels < 50%) for >=15m
- Business funnel panels show elevated drop-off

Quick checks:
1) Verify traffic exists: quotes, labels, shipments rates > 0 for affected tenant(s)
2) Check recent errors in gateway-api, label-svc, shipment-svc (Loki) with tenant filter
3) Inspect route SLO panels for elevated 5xx or latency

Possible causes:
- Label generation failures or downstream storage errors
- Shipment creation validation failures or DB issues
- Tenant-specific configuration or auth problems

Actions:
- Drill into traces for failing routes; validate span attributes tenant.id and app.shipment_id
- Check service health endpoints and deep health panels
- If widespread, consider rolling back recent changes affecting label/shipment flows

Metrics to watch:
- pricing_quotes_total, labels_generated_total, shipments_created_total (by tenant)
- http.server.errors and duration histograms for affected routes

