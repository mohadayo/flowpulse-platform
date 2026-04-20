import http from "http";

const PORT = parseInt(process.env.ANALYTICS_PORT || "8003", 10);
const LOG_LEVEL = process.env.LOG_LEVEL || "info";

interface AnalyticsRecord {
  id: number;
  metric: string;
  value: number;
  timestamp: string;
  dimensions: Record<string, string>;
}

interface DashboardSummary {
  totalRecords: number;
  metrics: Record<string, { count: number; sum: number; avg: number }>;
  lastUpdated: string;
}

const records: AnalyticsRecord[] = [];

function log(level: string, message: string): void {
  const levels = ["debug", "info", "warn", "error"];
  if (levels.indexOf(level) >= levels.indexOf(LOG_LEVEL)) {
    const ts = new Date().toISOString();
    console.log(`${ts} [${level.toUpperCase()}] analytics-api: ${message}`);
  }
}

function writeJSON(
  res: http.ServerResponse,
  status: number,
  data: unknown
): void {
  res.writeHead(status, { "Content-Type": "application/json" });
  res.end(JSON.stringify(data));
}

function readBody(req: http.IncomingMessage): Promise<string> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    req.on("data", (chunk: Buffer) => chunks.push(chunk));
    req.on("end", () => resolve(Buffer.concat(chunks).toString()));
    req.on("error", reject);
  });
}

function buildSummary(): DashboardSummary {
  const metrics: DashboardSummary["metrics"] = {};
  for (const r of records) {
    if (!metrics[r.metric]) {
      metrics[r.metric] = { count: 0, sum: 0, avg: 0 };
    }
    metrics[r.metric].count++;
    metrics[r.metric].sum += r.value;
    metrics[r.metric].avg = metrics[r.metric].sum / metrics[r.metric].count;
  }
  return {
    totalRecords: records.length,
    metrics,
    lastUpdated: new Date().toISOString(),
  };
}

export async function handleRequest(
  req: http.IncomingMessage,
  res: http.ServerResponse
): Promise<void> {
  const { method, url } = req;

  if (url === "/health" && method === "GET") {
    writeJSON(res, 200, {
      status: "healthy",
      service: "analytics-api",
      timestamp: Date.now(),
    });
    return;
  }

  if (url === "/analytics" && method === "POST") {
    try {
      const body = await readBody(req);
      const data = JSON.parse(body);
      if (!data.metric || data.value === undefined) {
        writeJSON(res, 400, {
          error: "analytics record must have 'metric' and 'value' fields",
        });
        return;
      }
      const record: AnalyticsRecord = {
        id: records.length + 1,
        metric: data.metric,
        value: Number(data.value),
        timestamp: new Date().toISOString(),
        dimensions: data.dimensions || {},
      };
      records.push(record);
      log("info", `Recorded metric=${record.metric} value=${record.value}`);
      writeJSON(res, 201, record);
    } catch {
      writeJSON(res, 400, { error: "invalid JSON" });
    }
    return;
  }

  if (url === "/analytics" && method === "GET") {
    writeJSON(res, 200, { records, count: records.length });
    return;
  }

  if (url === "/dashboard" && method === "GET") {
    writeJSON(res, 200, buildSummary());
    return;
  }

  writeJSON(res, 404, { error: "not found" });
}

export function createServer(): http.Server {
  return http.createServer(handleRequest);
}

export function getRecords(): AnalyticsRecord[] {
  return records;
}

export function clearRecords(): void {
  records.length = 0;
}

if (require.main === module) {
  const server = createServer();
  server.listen(PORT, "0.0.0.0", () => {
    log("info", `Analytics API starting on port ${PORT}`);
  });
}
