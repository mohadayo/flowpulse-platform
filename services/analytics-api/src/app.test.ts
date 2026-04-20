import http from "http";
import { createServer, clearRecords } from "./app";

let server: http.Server;
let baseUrl: string;

function request(
  path: string,
  options: { method?: string; body?: unknown } = {}
): Promise<{ status: number; body: Record<string, unknown> }> {
  return new Promise((resolve, reject) => {
    const url = new URL(path, baseUrl);
    const req = http.request(
      url,
      { method: options.method || "GET" },
      (res) => {
        const chunks: Buffer[] = [];
        res.on("data", (chunk: Buffer) => chunks.push(chunk));
        res.on("end", () => {
          const text = Buffer.concat(chunks).toString();
          resolve({
            status: res.statusCode || 0,
            body: JSON.parse(text),
          });
        });
      }
    );
    req.on("error", reject);
    if (options.body) {
      req.write(JSON.stringify(options.body));
    }
    req.end();
  });
}

beforeAll((done) => {
  server = createServer();
  server.listen(0, "127.0.0.1", () => {
    const addr = server.address() as { port: number };
    baseUrl = `http://127.0.0.1:${addr.port}`;
    done();
  });
});

afterAll((done) => {
  server.close(done);
});

beforeEach(() => {
  clearRecords();
});

test("GET /health returns healthy", async () => {
  const { status, body } = await request("/health");
  expect(status).toBe(200);
  expect(body.status).toBe("healthy");
  expect(body.service).toBe("analytics-api");
});

test("POST /analytics creates a record", async () => {
  const { status, body } = await request("/analytics", {
    method: "POST",
    body: { metric: "page_views", value: 42 },
  });
  expect(status).toBe(201);
  expect(body.metric).toBe("page_views");
  expect(body.value).toBe(42);
  expect(body.id).toBe(1);
});

test("POST /analytics with dimensions", async () => {
  const { status, body } = await request("/analytics", {
    method: "POST",
    body: { metric: "clicks", value: 10, dimensions: { page: "/home" } },
  });
  expect(status).toBe(201);
  expect((body.dimensions as Record<string, string>).page).toBe("/home");
});

test("POST /analytics rejects missing fields", async () => {
  const { status, body } = await request("/analytics", {
    method: "POST",
    body: { foo: "bar" },
  });
  expect(status).toBe(400);
  expect(body.error).toBeDefined();
});

test("POST /analytics rejects invalid JSON", async () => {
  return new Promise<void>((resolve, reject) => {
    const url = new URL("/analytics", baseUrl);
    const req = http.request(url, { method: "POST" }, (res) => {
      const chunks: Buffer[] = [];
      res.on("data", (chunk: Buffer) => chunks.push(chunk));
      res.on("end", () => {
        expect(res.statusCode).toBe(400);
        resolve();
      });
    });
    req.on("error", reject);
    req.write("not-json");
    req.end();
  });
});

test("GET /analytics returns records", async () => {
  await request("/analytics", {
    method: "POST",
    body: { metric: "views", value: 5 },
  });
  await request("/analytics", {
    method: "POST",
    body: { metric: "clicks", value: 3 },
  });
  const { status, body } = await request("/analytics");
  expect(status).toBe(200);
  expect(body.count).toBe(2);
});

test("GET /dashboard returns summary", async () => {
  await request("/analytics", {
    method: "POST",
    body: { metric: "views", value: 10 },
  });
  await request("/analytics", {
    method: "POST",
    body: { metric: "views", value: 20 },
  });
  const { status, body } = await request("/dashboard");
  expect(status).toBe(200);
  expect(body.totalRecords).toBe(2);
  const metrics = body.metrics as Record<
    string,
    { count: number; sum: number; avg: number }
  >;
  expect(metrics.views.count).toBe(2);
  expect(metrics.views.sum).toBe(30);
  expect(metrics.views.avg).toBe(15);
});

test("GET /nonexistent returns 404", async () => {
  const { status } = await request("/nonexistent");
  expect(status).toBe(404);
});
