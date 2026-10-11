import assert from "node:assert/strict";
import { execFile } from "node:child_process";
import { randomBytes } from "node:crypto";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import http from "node:http";
import https from "node:https";
import net from "node:net";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";

const execute = promisify(execFile);
const root = fileURLToPath(new URL("../", import.meta.url));
const project = `flowkit-production-check-${randomBytes(5).toString("hex")}`;
const caddy = "caddy:2.11.7-alpine@sha256:d8542f48d34a9cf4e4c11a478865229840e87e4c96ea3f439101f31a5d35f75f";
const password = randomBytes(24).toString("hex");
let sessionCookie = "";
const temporary = await mkdtemp(path.join(os.tmpdir(), `${project}-`));
const env = { ...process.env, OPENROUTER_API_KEY: "", FAL_KEY: "", FAL_BILLING_KEY: "",
  POSTGRES_PASSWORD: randomBytes(32).toString("hex"), FLOWKIT_DOMAIN: "flowkit.localhost",
  DASHBOARD_SESSION_KEY: randomBytes(32).toString("hex"), EDGE_BIND_ADDRESS: "127.0.0.1", LOAD_EXAMPLES: "true",
  WORKER_CONCURRENCY: "1", LOG_LEVEL: "info", COMPOSE_PROFILES: "" };
const argumentsBase = ["compose", "-p", project, "-f", "compose.yaml", "-f", "compose.production.yaml"];
let ca;
let runID = null;
let stackCreated = false;

function log(level, event, fields = {}) {
  console.log(JSON.stringify({ time: new Date().toISOString(), level, service: "production-check",
    run_id: runID, node_id: null, attempt: null, event, ...fields }));
}
async function docker(args) {
  return (await execute("docker", args, { cwd: root, env, timeout: 600000, maxBuffer: 8 * 1024 * 1024 })).stdout.trim();
}
const compose = (...args) => docker([...argumentsBase, ...args]);
const pause = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
async function freePort() {
  const server = net.createServer();
  await new Promise((resolve, reject) => server.once("error", reject).listen(0, "127.0.0.1", resolve));
  const port = server.address().port;
  await new Promise((resolve) => server.close(resolve));
  return String(port);
}
async function request(route, { authenticated = true, method = "GET", body, headers = {}, plaintext = false } = {}) {
  const data = body === undefined ? undefined : typeof body === "string" ? body : JSON.stringify(body);
  return new Promise((resolve, reject) => {
    const transport = plaintext ? http : https;
    const req = transport.request({ hostname: "flowkit.localhost", servername: "flowkit.localhost",
      port: Number(plaintext ? env.EDGE_HTTP_PORT : env.EDGE_HTTPS_PORT), path: route, method,
      ca, autoSelectFamily: false, lookup: (_name, _options, callback) => callback(null, "127.0.0.1", 4),
      headers: { Host: `flowkit.localhost:${plaintext ? env.EDGE_HTTP_PORT : env.EDGE_HTTPS_PORT}`, ...(authenticated && sessionCookie ? { Cookie: sessionCookie } : {}),
        ...(data ? { "Content-Type": "application/json", "Content-Length": Buffer.byteLength(data) } : {}), ...headers },
    }, (response) => {
      const chunks = [];
      response.on("data", (chunk) => chunks.push(chunk));
      response.on("error", reject);
      response.on("end", () => {
        const text = Buffer.concat(chunks).toString("utf8");
        try { resolve({ status: response.statusCode, headers: response.headers,
          data: response.headers["content-type"]?.includes("application/json") ? JSON.parse(text) : text }); }
        catch (error) { reject(error); }
      });
    });
    req.on("error", reject);
    req.setTimeout(5000, () => req.destroy(new Error("Local HTTPS check timed out")));
    req.end(data);
  });
}

try {
  env.EDGE_HTTP_PORT = await freePort();
  env.EDGE_HTTPS_PORT = await freePort();
  // This disposable password is not a real credential. No local .env provider key is inherited.
  env.DASHBOARD_PASSWORD_HASH = await docker(["run", "--rm", caddy, "caddy", "hash-password", "--plaintext", password]);
  const config = JSON.parse(await compose("--profile", "*", "config", "--format", "json"));
  for (const service of ["db", "api", "dashboard", "gate"]) assert.equal(config.services[service].ports?.length ?? 0, 0);
  assert(!("OPENROUTER_API_KEY" in config.services.gate.environment));
  assert(!("FAL_KEY" in config.services.gate.environment));
  assert.deepEqual(config.services.worker.profiles, ["generation"]);
  assert.equal(config.services.proxy.ports.length, 2);
  assert(config.services.proxy.ports.every((port) => port.host_ip === "127.0.0.1"));
  for (const service of ["init", "api", "bootstrap", "worker"]) {
    const settings = config.services[service].environment;
    for (const key of ["OPENROUTER_API_KEY", "FAL_KEY", "FAL_BILLING_KEY"]) assert.equal(settings[key], "");
    assert(settings.DATABASE_URL.includes(env.POSTGRES_PASSWORD));
  }
  stackCreated = true;
  log("INFO", "production.starting", { project });
  await compose("up", "--build", "-d", "--wait", "--wait-timeout", "180");
  assert.equal(await compose("ps", "-q", "worker"), "", "Default production startup must not launch the worker");
  const proxy = await compose("ps", "-q", "proxy");
  const certificate = path.join(temporary, "root.crt");
  for (let attempt = 0; ; attempt++) {
    try { await docker(["cp", `${proxy}:/data/caddy/pki/authorities/local/root.crt`, certificate]); break; }
    catch (error) { if (attempt >= 20) throw error; await pause(500); }
  }
  ca = await readFile(certificate);
  // Verify the local CA normally; never disable certificate validation.
  const redirect = await request("/", { plaintext: true, authenticated: false });
  assert.equal(redirect.status, 308);
  assert.match(redirect.headers.location, /^https:\/\/flowkit\.localhost/);
  const rootRedirect = await request("/", { authenticated: false });
  assert.equal(rootRedirect.status, 303);
  assert.equal(rootRedirect.headers.location, "/auth/login");
  const loginPage = await request("/auth/login", { authenticated: false });
  assert.equal(loginPage.status, 200);
  assert.equal((loginPage.data.match(/<input/g) ?? []).length, 1);
  assert(!loginPage.data.includes("username"));
  for (const route of ["/models", "/workflows", "/runs", "/health/ready", "/artifacts/test"]) {
    assert.equal((await request(route, { authenticated: false })).status, 401, route);
  }
  assert.equal((await request("/workflows", { authenticated: false, method: "POST", body: {} })).status, 401);
  assert.equal((await request("/models", { authenticated: false, headers: { Authorization: "Basic invalid" } })).status, 401);
  const origin = `https://flowkit.localhost:${env.EDGE_HTTPS_PORT}`;
  const loginOptions = (value, requestOrigin = origin) => ({ authenticated: false, method: "POST",
    body: new URLSearchParams({ password: value }).toString(),
    headers: { "Content-Type": "application/x-www-form-urlencoded", Origin: requestOrigin } });
  const wrongPassword = await request("/auth/login", loginOptions("wrong-password"));
  assert.equal(wrongPassword.status, 401);
  assert.equal(wrongPassword.headers["www-authenticate"], undefined);
  assert.equal(wrongPassword.headers["set-cookie"], undefined);
  assert.equal((await request("/auth/login", loginOptions(password, "https://attacker.test"))).status, 403);
  async function login() {
    const response = await request("/auth/login", loginOptions(password));
    assert.equal(response.status, 303);
    assert.equal(response.headers.location, "/");
    const cookie = response.headers["set-cookie"]?.[0];
    assert(cookie);
    for (const flag of ["Secure", "HttpOnly", "SameSite=Strict", "Path=/", "Max-Age=43200"]) assert(cookie.includes(flag));
    assert(!cookie.includes("Domain="));
    sessionCookie = cookie.split(";")[0];
  }
  await login();
  assert.equal((await request("/auth/session")).data.authenticated, true);
  assert.equal((await request("/workflows", { method: "POST", body: {}, headers: { Origin: "https://attacker.test" } })).status, 403);
  assert.equal((await request("/models", { headers: { Cookie: `${sessionCookie}tampered` } })).status, 401);
  assert.equal((await request("/health/ready")).status, 200);
  const page = await request("/");
  assert.equal(page.status, 200);
  assert.match(page.data, /<title>FlowKit/);
  const asset = page.data.match(/src="(\/assets\/[^\"]+\.js)"/);
  assert(asset);
  assert.equal((await request(asset[1], { authenticated: false })).status, 401);
  assert.equal((await request(asset[1])).status, 200);
  assert.equal((await request("/models")).data.items.length, 3);
  assert.equal((await request("/workflows")).data.items.length, 2);
  const fixture = { name: "Production providerless smoke", definition: { schemaVersion: 1,
    nodes: [{ id: "input", kind: "input", params: { textField: "message", inputSchema: {
      type: "object", properties: { message: { type: "string", minLength: 1 } },
      required: ["message"], additionalProperties: false } } }, { id: "output", kind: "output" }],
    edges: [{ from: "input", fromPort: "text", to: "output", toPort: "text" }] } };
  const created = await request("/workflows", { method: "POST", body: fixture });
  assert.equal(created.status, 201);
  const runPath = `/workflows/${created.data.id}/runs`;
  const options = { method: "POST", body: { input: { message: "TLS worker check" } },
    headers: { "Idempotency-Key": project } };
  const accepted = await request(runPath, options);
  assert.equal(accepted.status, 202);
  runID = accepted.data.run_id;
  assert.equal((await request(`/runs/${runID}`)).data.status, "queued");
  await compose("up", "-d", "worker"); // Explicit opt-in, with empty credentials and a providerless graph.
  let run;
  for (let attempt = 0; attempt < 50; attempt++) {
    run = (await request(`/runs/${runID}`)).data;
    if (["succeeded", "failed"].includes(run.status)) break;
    await pause(400);
  }
  assert.equal(run.status, "succeeded");
  assert.equal(run.output.text, "TLS worker check");
  assert(run.nodes.every((node) => node.provider_requests.length === 0));
  assert.equal(run.cost.estimated_cost_usd, "0.000000000000");
  assert.equal(run.cost.cost_status, "complete");
  const replay = await request(runPath, options);
  assert.equal(replay.status, 200);
  assert.equal(replay.data.run_id, runID);
  assert.equal((await request(`/workflows/${created.data.id}`, { method: "DELETE" })).status, 204);
  await compose("restart", "gate", "proxy");
  for (let attempt = 0; ; attempt++) {
    try { assert.equal((await request("/health/ready")).status, 200); break; }
    catch (error) { if (attempt >= 20) throw error; await pause(500); }
  }
  assert.equal((await request("/models", { authenticated: false })).status, 401);
  assert.equal((await request("/auth/logout", { method: "POST", headers: { Origin: "https://attacker.test" } })).status, 403);
  const logout = await request("/auth/logout", { method: "POST", headers: { Origin: origin } });
  assert.equal(logout.status, 303);
  assert(logout.headers["set-cookie"][0].includes("Max-Age=0"));
  sessionCookie = "";
  assert.equal((await request("/models")).status, 401);
  await login();
  const capturedLogs = await execute("docker", ["logs", proxy], { cwd: root, env, maxBuffer: 8 * 1024 * 1024 });
  const logs = capturedLogs.stdout + capturedLogs.stderr;
  assert(!logs.includes(password) && !logs.includes(sessionCookie.split("=")[1]));
  const proxyLogs = logs.split(/\r?\n/).filter(Boolean).map((line) => JSON.parse(line));
  const accessLogs = proxyLogs.filter((entry) => entry.logger?.startsWith("http.log.access"));
  assert(accessLogs.length > 0);
  for (const entry of proxyLogs) {
    assert.equal(entry.service, "edge");
    for (const key of ["run_id", "node_id", "attempt"]) assert.equal(entry[key], null);
  }
  const gateLogs = await compose("logs", "--no-color", "--no-log-prefix", "gate");
  assert(!gateLogs.includes(password) && !gateLogs.includes(sessionCookie.split("=")[1]) && !gateLogs.includes(env.DASHBOARD_SESSION_KEY));
  for (const line of gateLogs.split(/\r?\n/).filter(Boolean)) {
    const entry = JSON.parse(line);
    for (const key of ["run_id", "node_id", "attempt"]) assert.equal(entry[key], null);
  }
  log("INFO", "production.checked", { tls: "verified-local-ca", authentication: "password-only-signed-session-all-routes",
    worker: "opt-in", restart: "passed", provider_calls: 0, access_log_lines: accessLogs.length });
} catch (error) {
  // Child-process errors can include arguments/environment; only expose a sanitized message.
  const sanitize = (text) => text.replaceAll(password, "[redacted]").replaceAll(sessionCookie || "unused-placeholder", "[redacted]").replaceAll(env.DASHBOARD_SESSION_KEY, "[redacted]").replaceAll(env.DASHBOARD_PASSWORD_HASH || "unused-hash", "[redacted]");
  const message = sanitize(error.message);
  log("ERROR", "production.check_failed", { error: message.slice(-2200) });
  if (stackCreated) {
    const logs = await compose("logs", "--no-color", "--tail=15").catch(() => "");
    log("ERROR", "production.failure_logs", { logs: sanitize(logs).slice(-4500) });
  }
  process.exitCode = 1;
} finally {
  if (stackCreated) await compose("--profile", "*", "down", "--volumes", "--remove-orphans").catch(() => {
    log("ERROR", "production.cleanup_failed", { project }); process.exitCode = 1;
  });
  assert(temporary.startsWith(path.join(os.tmpdir(), `${project}-`)));
  await rm(temporary, { recursive: true, force: true });
}
